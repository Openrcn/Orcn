package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"orcn/core"
	"orcn/core/config"
	"orcn/core/logger"
	"orcn/models"
	"orcn/providers/local"
	"orcn/providers/nosana"
	"orcn/services/controller"
)

func main() {
	log := logger.New("CONTROLLER")
	log.Info("Starting orcn Controller...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize DB
	db, err := models.InitDB(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal("Failed to init database: %v", err)
	}
	log.Info("Database initialized successfully.")

	// Register Providers (needed for reconciliation)
	nosClient := nosana.New()
	core.RegisterProvider("nosana", nosClient)

	localClient := local.New()
	core.RegisterProvider("local", localClient)

	log.Info("Providers registered successfully.")

	// Start Background Controller Loop
	ctrl := controller.New(db, cfg)
	ctrl.StartLoop()

	log.Info("Controller running. Press Ctrl+C to stop.")
	<-ctx.Done()
	log.Info("Shutting down...")
	time.Sleep(500 * time.Millisecond)
	log.Info("Shutdown complete")
}