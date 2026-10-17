package main

import (
	"context"
	"os/signal"
	"strings"
	"syscall"

	"orcn/core/logger"
	"orcn/services/node-agent/agent"
	"orcn/services/node-agent/config"
	"orcn/services/node-agent/engines/docker"
	"orcn/services/node-agent/events"
	"orcn/services/node-agent/health"
	restinterface "orcn/services/node-agent/interfaces/rest"
	"orcn/services/node-agent/telemetry"
)

func main() {
	cfg := config.Load()
	logger.SetLevel(logLevel(cfg.LogLevel))
	log := logger.New("NODE-AGENT")

	// 1. Initialize Event System
	eventBuffer := events.NewBuffer(500)
	recorder := events.NewRecorder(eventBuffer, log)

	// 2. Initialize Container Engine (driver dependency)
	engine, err := docker.NewEngine(cfg, recorder)
	if err != nil {
		log.Fatal("Failed to initialize Docker Engine: %v", err)
	}
	defer engine.Close()

	// 3. Initialize Background Workers (Telemetry & Health)
	var telemetryBuffer *telemetry.Buffer
	var telemetryCollector *telemetry.Collector
	if cfg.TelemetryEnabled {
		telemetryCollector = telemetry.NewCollector(cfg.TelemetryInterval, cfg.TelemetryBufferPoints)
		telemetryBuffer = telemetryCollector.Buffer()
	}

	// 4. Initialize Network Interfaces (Admin Control Plane & Proxy Router)
	//    State is created here so the health monitor can reference it.
	state := restinterface.NewRegistrationState(cfg.AgentMode, cfg.MaxWorkloadCount)

	var healthStore *health.Store
	var healthMonitor *health.Monitor
	if cfg.HealthCheckEnabled {
		healthStore = health.NewStore()
		healthMonitor = health.NewMonitor(health.Config{
			Interval:         cfg.HealthCheckInterval,
			FailureThreshold: cfg.HealthCheckFailureThreshold,
			ProxyTargetHost:  cfg.ProxyTargetHost,
		}, state, engine, healthStore)
	}

	adminServer, proxyServer := restinterface.InitServers(cfg, engine, eventBuffer, telemetryBuffer, healthStore, state)

	// 5. Register Components into Orchestrator
	nodeAgent := agent.New()
	if telemetryCollector != nil {
		nodeAgent.Register(telemetryCollector)
	}
	if healthMonitor != nil {
		nodeAgent.Register(healthMonitor)
	}
	nodeAgent.Register(adminServer)
	nodeAgent.Register(proxyServer)

	// 6. Boot Node Agent with Signal Handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("Booting node-agent components...")
	if err := nodeAgent.Start(ctx); err != nil {
		log.Fatal("Agent crashed: %v", err)
	}
}

func logLevel(value string) logger.LogLevel {
	switch strings.ToLower(value) {
	case "debug":
		return logger.LevelDebug
	case "warn", "warning":
		return logger.LevelWarn
	case "error":
		return logger.LevelError
	case "none", "off":
		return logger.LevelNone
	default:
		return logger.LevelInfo
	}
}
