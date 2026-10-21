.PHONY: all build-gateway build-controller build-node-agent build-resource-loader run-gateway run-controller run-node-agent clean docker-node-agent

# Default target
all: build-gateway build-controller build-node-agent build-resource-loader

# ==========================================
# Gateway Service
# ==========================================

build-gateway:
	@echo "Building Gateway Service..."
	@mkdir -p bin
	@go build -o bin/gateway ./cmd/gateway
	@echo "Build complete! Binary located at: bin/gateway"

run-gateway:
	@echo "Starting Gateway Service..."
	@go run ./cmd/gateway

# ==========================================
# Controller Service
# ==========================================

build-controller:
	@echo "Building Controller Service..."
	@mkdir -p bin
	@go build -o bin/controller ./cmd/controller
	@echo "Build complete! Binary located at: bin/controller"

run-controller:
	@echo "Starting Controller Service..."
	@go run ./cmd/controller

# ==========================================
# Node Agent
# ==========================================

build-node-agent:
	@echo "Building Node Agent..."
	@mkdir -p bin
	@go build -o bin/node-agent ./cmd/node-agent
	@echo "Build complete! Binary located at: bin/node-agent"

run-node-agent:
	@if [ "$(DETACH)" = "1" ]; then \
		echo "Starting Node Agent (background)..."; \
		NODE_AGENT_ADMIN_ADDRESS=127.0.0.1:9090 \
		NODE_AGENT_PROXY_ADDRESS=0.0.0.0:8081 \
		NODE_AGENT_ADMIN_TOKEN=orcn-dev-admin-token-2024 \
		AGENT_MODE=shared \
		MAX_WORKLOAD_COUNT=10 \
		NODE_AGENT_TELEMETRY_ENABLED=true \
		NODE_AGENT_HEALTHCHECK_ENABLED=true \
		NODE_AGENT_LOG_LEVEL=info \
		go run ./cmd/node-agent > /tmp/node-agent.log 2>&1 & echo $$! > /tmp/node-agent.pid; \
		echo "Node Agent started in background (PID: $$(cat /tmp/node-agent.pid))"; \
		echo "Admin Token: orcn-dev-admin-token-2024"; \
		echo "Admin: http://127.0.0.1:9090"; \
		echo "Proxy: http://127.0.0.1:8081"; \
		echo "Logs: tail -f /tmp/node-agent.log"; \
	else \
		echo "Starting Node Agent (foreground)..."; \
		NODE_AGENT_ADMIN_ADDRESS=127.0.0.1:9090 \
		NODE_AGENT_PROXY_ADDRESS=0.0.0.0:8081 \
		NODE_AGENT_ADMIN_TOKEN=orcn-dev-admin-token-2024 \
		AGENT_MODE=shared \
		MAX_WORKLOAD_COUNT=10 \
		NODE_AGENT_TELEMETRY_ENABLED=true \
		NODE_AGENT_HEALTHCHECK_ENABLED=true \
		NODE_AGENT_LOG_LEVEL=info \
		go run ./cmd/node-agent; \
	fi

stop-node-agent:
	@if [ -f /tmp/node-agent.pid ]; then kill $$(cat /tmp/node-agent.pid) && rm /tmp/node-agent.pid && echo "Node Agent stopped"; else echo "No PID file found"; fi

# ==========================================
# Resource Loader (sidecar for node-agent)
# ==========================================

build-resource-loader:
	@echo "Building Resource Loader..."
	@mkdir -p bin
	@go build -o bin/resource-loader ./services/node-agent/resource-loader
	@echo "Build complete! Binary located at: bin/resource-loader"

# ==========================================
# Docker
# ==========================================

docker-node-agent:
	@echo "Building Node Agent Docker image..."
	@docker build -t orcn/node-agent:latest -f services/node-agent/Dockerfile .
	@echo "Docker image built: orcn/node-agent:latest"

docker-gateway:
	@echo "Building Gateway Docker image..."
	@docker build -t orcn/gateway:latest -f services/gateway/Dockerfile .
	@echo "Docker image built: orcn/gateway:latest"

run-node-agent-docker:
	@echo "Running Node Agent Docker container..."
	@docker run -d \
		--name orcn-node-agent \
		--restart unless-stopped \
		-p 9090:9090 \
		-p 8081:8080 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e NODE_AGENT_ADMIN_ADDRESS=0.0.0.0:9090 \
		-e NODE_AGENT_PROXY_ADDRESS=0.0.0.0:8080 \
		-e NODE_AGENT_ADMIN_TOKEN=orcn-dev-admin-token-2024 \
		-e AGENT_MODE=shared \
		-e MAX_WORKLOAD_COUNT=10 \
		-e NODE_AGENT_TELEMETRY_ENABLED=true \
		-e NODE_AGENT_HEALTHCHECK_ENABLED=true \
		-e NODE_AGENT_LOG_LEVEL=info \
		orcn/node-agent:latest
	@docker network connect infra_orcn-net orcn-node-agent 2>/dev/null || true
	@echo "Node Agent container started: orcn-node-agent"
	@echo "Admin Token: orcn-dev-admin-token-2024"
	@echo "Admin: http://127.0.0.1:9090"
	@echo "Proxy: http://127.0.0.1:8081"

stop-node-agent-docker:
	@docker stop orcn-node-agent && docker rm orcn-node-agent
	@echo "Node Agent container stopped and removed"

logs-node-agent-docker:
	@docker logs -f orcn-node-agent

# ==========================================
# Utilities
# ==========================================

clean:
	@echo "Cleaning up bin directory..."
	@rm -rf bin/
	@echo "Clean complete."