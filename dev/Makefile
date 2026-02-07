# FRR Exporter Development Environment
# =====================================

SHELL := /bin/bash
COMPOSE := docker compose
FRR_VERSION ?= 10.5.1
TEST_VERSIONS ?= 7.5.1 8.5.7 10.5.1

# Exporter binary location (use /tmp for Lima VM compatibility)
EXPORTER_BIN := /tmp/frr_exporter_linux

.PHONY: help up down build deploy metrics metrics2 vtysh1 vtysh2 status bgp ospf bfd pim vrrp routes test-all-versions clean \
        lima-setup lima-provision lima-shell lima-start lima-stop lima-delete

help:
	@echo "FRR Exporter Development Environment"
	@echo "===================================="
	@echo ""
	@echo "Quick Start:"
	@echo "  make up                              - Start FRR routers (FRR $(FRR_VERSION))"
	@echo "  make build                           - Build frr_exporter"
	@echo "  make deploy                          - Deploy exporter to containers"
	@echo "  make redeploy                        - Rebuild and redeploy exporter"
	@echo "  make down                            - Stop environment"
	@echo "  make restart                         - Restart environment"
	@echo "  make clean                           - Stop and remove built binary"
	@echo ""
	@echo "Or all at once:"
	@echo "  make up build deploy                 - Start everything"
	@echo ""
	@echo "Different FRR version:"
	@echo "  make up FRR_VERSION=7.5.1"
	@echo ""
	@echo "Metrics:"
	@echo "  make metrics1                        - View metrics from router1"
	@echo "  make metrics2                        - View metrics from router2"
	@echo ""
	@echo "Debugging:"
	@echo "  make vtysh1                          - Open vtysh on router1"
	@echo "  make vtysh2                          - Open vtysh on router2"
	@echo "  make status                          - Show all protocol status"
	@echo "  make show-bgp                        - Show BGP status"
	@echo "  make show-ospf                       - Show OSPF status"
	@echo "  make show-bfd                        - Show BFD status"
	@echo "  make show-pim                        - Show PIM status"
	@echo "  make show-vrrp                       - Show VRRP status"
	@echo "  make show-routes                     - Show route summary"
	@echo "  make show-version                    - Show FRR version"
	@echo "  make logs                            - View FRR container logs"
	@echo "  make exporter-logs                   - View exporter logs on router1 and router2"
	@echo "  make exporter-logs-follow-router1    - Tail exporter logs on router1"
	@echo "  make exporter-logs-follow-router2    - Tail exporter logs on router2"
	@echo ""
	@echo "Testing different versions:"
	@echo "  make test-all-versions TEST_VERSIONS=\"8.5.7 10.5.1\""
	@echo ""
	@echo "macOS Setup (VRF requires Lima VM):"
	@echo "  make lima-setup                      - Create Ubuntu VM"
	@echo "  make lima-provision                  - Install Docker and tools in VM"
	@echo "  make lima-shell                      - Shell into VM"
	@echo "  make lima-start                      - Start the VM"
	@echo "  make lima-stop                       - Stop the VM"
	@echo "  make lima-delete                     - Remove the VM"
	@echo ""
	@echo "Endpoints (after deploy):"
	@echo "  http://localhost:9342/metrics        - Router1"
	@echo "  http://localhost:9343/metrics        - Router2"

# Start FRR containers
up:
	@echo "Starting FRR $(FRR_VERSION) routers:"
	FRR_VERSION=$(FRR_VERSION) $(COMPOSE) up -d
	@echo "Sleeping to ensure containers have started:"
	@sleep 5
	@$(COMPOSE) ps
	@echo ""
	@echo "FRR ready:"
	@echo "  Router1: make vtysh1"
	@echo "  Router2: make vtysh2"
	@echo "Start exporter: make build deploy"

# Stop all containers
down:
	$(COMPOSE) down
	@echo "Stopped."

# Restart
restart: down up

# Build frr_exporter
build:
	@echo "Building frr_exporter for Linux:"
	cd .. && CGO_ENABLED=0 go build -o $(EXPORTER_BIN) .
	@echo "Built: $(EXPORTER_BIN)"

# Deploy exporter to containers and start it
deploy: $(EXPORTER_BIN)
	@echo "Deploying exporter to containers:"
	docker cp $(EXPORTER_BIN) frr-router1:/usr/local/bin/frr_exporter
	docker cp $(EXPORTER_BIN) frr-router2:/usr/local/bin/frr_exporter
	-docker exec frr-router1 pkill -f frr_exporter 2>/dev/null || true
	-docker exec frr-router2 pkill -f frr_exporter 2>/dev/null || true
	@sleep 1
	docker exec -d frr-router1 sh -c '/usr/local/bin/frr_exporter \
		--collector.bgp \
		--collector.bgp6 \
		--collector.ospf \
		--collector.bfd \
		--collector.pim \
		--collector.vrrp \
		--collector.route \
		--collector.route.detailed-routes \
		--collector.bgp.peer-descriptions \
		--web.listen-address=:9342 \
		2>&1 | tee /var/log/frr_exporter.log'
	docker exec -d frr-router2 sh -c '/usr/local/bin/frr_exporter \
		--collector.bgp \
		--collector.bgp6 \
		--collector.ospf \
		--collector.bfd \
		--collector.pim \
		--collector.vrrp \
		--collector.route \
		--collector.route.detailed-routes \
		--collector.bgp.peer-descriptions \
		--web.listen-address=:9342 \
		2>&1 | tee /var/log/frr_exporter.log'
	@echo ""
	@echo "Exporters started:"
	@echo "  Router1: http://localhost:9342/metrics"
	@echo "  Router2: http://localhost:9343/metrics"
	@echo ""
	@echo "View logs: make exporter-logs"

$(EXPORTER_BIN):
	@echo "Exporter binary not found. Building:"
	$(MAKE) build

# Redeploy (rebuild and deploy)
redeploy: build deploy

# View FRR logs
logs:
	$(COMPOSE) logs -f

logs-router1:
	$(COMPOSE) logs -f router1

logs-router2:
	$(COMPOSE) logs -f router2

# View exporter logs
exporter-logs:
	@echo "=== Router1 Exporter Logs ==="
	@docker exec frr-router1 cat /var/log/frr_exporter.log
	@echo ""
	@echo "=== Router2 Exporter Logs ==="
	@docker exec frr-router2 cat /var/log/frr_exporter.log 

exporter-logs-follow-router1:
	docker exec frr-router1 tail -f /var/log/frr_exporter.log

exporter-logs-follow-router2:
	docker exec frr-router2 tail -f /var/log/frr_exporter.log

# Fetch metrics from router1
metrics1:
	@echo "=== FRR Exporter Metrics (Router1) ==="
	@curl -s http://localhost:9342/metrics
	@echo ""

# Fetch metrics from router2
metrics2:
	@echo "=== FRR Exporter Metrics (Router2) ==="
	@curl -s http://localhost:9343/metrics
	@echo ""


# Open vtysh on router1
vtysh1:
	docker exec -it frr-router1 vtysh

# Open vtysh on router2
vtysh2:
	docker exec -it frr-router2 vtysh

# Show overall status
status: show-bgp show-ospf show-bfd show-pim show-vrrp

# Show BGP status
show-bgp:
	@echo "=== BGP Status (Router1) ==="
	@docker exec frr-router1 vtysh -c "show bgp summary" 
	@echo ""
	@echo "=== BGP VRF vrf-red Status ==="
	@docker exec frr-router1 vtysh -c "show bgp vrf vrf-red summary"

# Show OSPF status
show-ospf:
	@echo "=== OSPF Neighbors (Router1) ==="
	@docker exec frr-router1 vtysh -c "show ip ospf neighbor"
	@echo "=== OSPF Neighbors (Router2) ==="
	@docker exec frr-router2 vtysh -c "show ip ospf neighbor"

# Show BFD status
show-bfd:
	@echo "=== BFD Peers (Router1) ==="
	@docker exec frr-router1 vtysh -c "show bfd peers"
	@echo "=== BFD Peers (Route2) ==="
	@docker exec frr-router2 vtysh -c "show bfd peers"

# Show PIM status
show-pim:
	@echo "=== PIM Neighbors (Router1) ==="
	@docker exec frr-router1 vtysh -c "show ip pim neighbor"
	@echo "=== PIM Neighbors (Router2) ==="
	@docker exec frr-router2 vtysh -c "show ip pim neighbor"

# Show VRRP status
show-vrrp:
	@echo "=== VRRP Status (Router1) ==="
	@docker exec frr-router1 vtysh -c "show vrrp" 2>/dev/null

# Show routes
show-routes:
	@echo "=== IP Routes (Router1) ==="
	@docker exec frr-router1 vtysh -c "show ip route summary"
	@echo "=== IP Routes (Router2) ==="
	@docker exec frr-router2 vtysh -c "show ip route summary"

# Show FRR version
show-version:
	@echo "=== Version (Router1) ==="
	@docker exec frr-router1 vtysh -c "show version"
	@echo "=== Version (Router2) ==="
	@docker exec frr-router2 vtysh -c "show version"

# Test all supported FRR versions
test-all-versions: build
	@echo "Testing FRR Exporter with multiple FRR versions:"
	@echo "=================================================="
	@for version in $(TEST_VERSIONS); do \
		echo ""; \
		echo "=== Testing FRR $$version ==="; \
		echo "----------------------------"; \
		$(MAKE) down 2>/dev/null || true; \
		$(MAKE) up FRR_VERSION=$$version; \
		sleep 10; \
		$(MAKE) deploy; \
		sleep 3; \
		echo ""; \
		echo "Checking metrics on router1:"; \
		curl -s http://localhost:9342/metrics 
		echo "Checking metrics on router2:"; \
		curl -s http://localhost:9343/metrics 
	done
	@echo ""
	@echo "All version tests complete!"
	@$(MAKE) down

# =============================================================================
# macOS Lima VM Setup (required for VRF support)
# =============================================================================
# VRF requires CONFIG_NET_VRF kernel support, which is not available in Docker
# Desktop or Colima. Lima with Ubuntu provides a full Linux kernel with VRF.

LIMA_VM := frrdev

# Check if running on macOS
IS_MACOS := $(shell uname -s | grep -q Darwin && echo 1 || echo 0)

# Create Lima VM (does not provision - use lima-provision for that)
lima-setup:
ifeq ($(IS_MACOS),0)
	@echo "Lima is only needed on macOS. On Linux, use Docker directly."
else
	@if ! command -v limactl &> /dev/null; then \
		echo "Error: Lima not found. Install with: brew install lima"; \
		exit 1; \
	fi
	@if limactl list 2>/dev/null | grep -q "$(LIMA_VM).*Running"; then \
		echo "VM '$(LIMA_VM)' is already running."; \
	elif limactl list 2>/dev/null | grep -q "$(LIMA_VM)"; then \
		limactl start $(LIMA_VM); \
	else \
		echo "Creating Ubuntu VM '$(LIMA_VM)':"; \
		limactl start --tty=false --name=$(LIMA_VM) template:ubuntu-24.04; \
		echo ""; \
		echo "VM created. Now run 'make lima-provision' to install Docker and tools."; \
	fi
endif

# Install Docker, Go, and tools inside the Lima VM
GO_VERSION := 1.25.0

lima-provision:
	@echo "Provisioning Lima VM with Docker, Go, and tools:"
	limactl shell $(LIMA_VM) -- sudo apt-get update
	limactl shell $(LIMA_VM) -- sudo apt-get install -y \
		docker.io \
		docker-compose-v2 \
		make \
		curl \
		git
	@echo "Installing VRF kernel module:"
	-limactl shell $(LIMA_VM) -- bash -c "sudo apt-get install -y linux-modules-extra-\$$(uname -r)"
	-limactl shell $(LIMA_VM) -- sudo modprobe vrf
	echo "Starting Docker:"
	limactl shell $(LIMA_VM) -- sudo systemctl enable docker
	limactl shell $(LIMA_VM) -- sudo systemctl start docker
	limactl shell $(LIMA_VM) -- sudo usermod -aG docker $(shell whoami)
	@echo "Installing Go $(GO_VERSION):"
	limactl shell $(LIMA_VM) -- curl -fsSL https://go.dev/dl/go$(GO_VERSION).linux-arm64.tar.gz -o /tmp/go.tar.gz
	limactl shell $(LIMA_VM) -- sudo rm -rf /usr/local/go
	limactl shell $(LIMA_VM) -- sudo tar -C /usr/local -xzf /tmp/go.tar.gz
	limactl shell $(LIMA_VM) -- rm /tmp/go.tar.gz
	limactl shell $(LIMA_VM) -- bash -c "echo 'export PATH=\$$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/golang.sh >/dev/null"
	limactl shell $(LIMA_VM) -- sudo systemctl start docker.service
	@echo "Rebooting to apply settings:"	
	@limactl stop $(LIMA_VM)
	@limactl start $(LIMA_VM)
	@echo ""
	@echo "========================================"
	@echo "Lima VM provisioned:"
	@echo ""
	@echo "Run: make lima-shell"
	@echo "Then: make up build deploy"
	@echo "========================================"

# Shell into Lima VM
lima-shell:
	@limactl shell $(LIMA_VM)

# Start Lima VM
lima-start:
	@limactl start $(LIMA_VM)

# Stop Lima VM
lima-stop:
	@limactl stop $(LIMA_VM)

# Delete Lima VM
lima-delete:
	@limactl delete $(LIMA_VM)

# =============================================================================
# Cleanup
# =============================================================================

# Clean up
clean: down
	rm -f $(EXPORTER_BIN)
	@echo "Cleaned."
