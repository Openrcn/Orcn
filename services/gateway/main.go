package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"orcn/core"
	"orcn/core/config"
	"orcn/core/logger"
	"orcn/models"
	"orcn/providers/local"
	"orcn/providers/nosana"
	"orcn/runtimes/ollama"
	"orcn/runtimes/vllm"
	"orcn/services/controller"
	"orcn/services/gateway/api"
	"orcn/services/gateway/health"
	"orcn/services/gateway/ingress"
)

func main() {
	log := logger.New("GATEWAY")
	log.Info("Starting orcn Gateway...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config: %v", err)
	}
	log.Info("Loaded Config: Domain=%s, APIPort=%d, IngressPort=%d", cfg.AppDomain, cfg.APIPort, cfg.IngressPort)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. Initialize DB
	db, err := models.InitDB(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal("Failed to init database: %v", err)
	}
	log.Info("Database initialized successfully.")

	// 2. Register Plugins
	nosClient := nosana.New()
	core.RegisterProvider("nosana", nosClient)

	localClient := local.New()
	core.RegisterProvider("local", localClient)

	vllmRT := vllm.New()
	core.RegisterRuntime("vllm", vllmRT)

	ollamaRT := ollama.New()
	core.RegisterRuntime("ollama", ollamaRT)

	log.Info("Plugins registered successfully.")

	// 3. Create shared services
	healthChecker := health.New(db)

	// 4. Start Background Controller Loop (Control Plane)
	ctrl := controller.New(db, cfg)
	ctrl.StartLoop()

	// 5. Start the Admin API Server (Control Plane)
	apiServer := api.New(db, healthChecker, cfg)
	apiSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.APIPort),
		Handler: apiServer.Handler(),
	}
	go func() {
		log.Info("Admin API listening on :%d", cfg.APIPort)
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("API Server failed: %v", err)
		}
	}()

	internalAPI := fmt.Sprintf("http://127.0.0.1:%d/api/v1/internal/routes", cfg.APIPort)
	ingressServer := ingress.New(internalAPI, cfg)
	ingressSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.IngressPort),
		Handler: ingressServer.Handler(),
	}

	go func() {
		log.Info("Public AI Ingress listening on :%d", cfg.IngressPort)
		if err := ingressSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Ingress Server failed (did you forget sudo?): %v", err)
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := apiSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("API server shutdown error: %v", err)
	}
	if err := ingressSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("Ingress server shutdown error: %v", err)
	}
	log.Info("Shutdown complete")
}


