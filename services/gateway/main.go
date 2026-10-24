package main

import (
	"context"
	"net/http"
	"os/signal"
	"strconv"
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
	"orcn/services/gateway/api"
	"orcn/services/gateway/health"
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

	// 4. Start the Gateway API Server. Reconciliation is owned by cmd/controller
	// and public traffic is owned by cmd/ingress.
	apiServer := api.New(db, healthChecker, cfg)
	apiSrv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.APIPort),
		Handler: apiServer.Handler(),
	}
	go func() {
		log.Info("Admin API listening on :%d", cfg.APIPort)
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("API Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := apiSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("API server shutdown error: %v", err)
	}
	log.Info("Shutdown complete")
}
