package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"orcn/core/config"
	"orcn/core/logger"
	"orcn/services/ingress"
)

func main() {
	log := logger.New("INGRESS")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	routesURL := strings.TrimRight(cfg.ControlPlaneURL, "/") + "/api/v1/internal/routes"
	server := ingress.New(routesURL, ingress.ConfigFromAppConfig(cfg))
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.IngressPort),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("Public ingress listening on :%d; control plane=%s", cfg.IngressPort, cfg.ControlPlaneURL)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Ingress server failed: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("Ingress shutdown error: %v", err)
	}
}
