# FRR Exporter Development Environment

Docker-based environment for testing frr_exporter against real FRR instances. Also includes setting up a Ubuntu VM using Lima on MacOS for VRF support.

## Quick Start

```bash
# Start FRR routers, build exporter, and deploy
make up build deploy

# View metrics
make metrics1
make metrics2

# Check protocol status
make status

# Stop
make down
```

### MacOS
```bash
# Create Ubuntu VM
make lima-setup

# Install Docker, make, and tools
make lima-provision   

# Shell into the VM (your mac filesystem is mounted)
make lima-shell
```

## Testing Different FRR Versions

```bash
make up FRR_VERSION=7.5.1 && make deploy
make up FRR_VERSION=8.5.7 && make deploy
make up FRR_VERSION=10.5.1 && make deploy  # default

# Test specific versions
make test-all-versions TEST_VERSIONS="8.5.7 10.5.1"
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  ┌──────────────┐     eth0 (10.0.0.0/24)  ┌──────────────┐     │
│  │   Router1    │◄───────────────────────►│   Router2    │     │
│  │  AS 65001    │    BGP/OSPF/BFD/PIM     │  AS 65002    │     │
│  │  10.0.0.10   │         VRRP            │  10.0.0.11   │     │
│  │  :9342       │                         │  :9343       │     │
│  │              │◄───────────────────────►│              │     │
│  │  10.1.0.10   │  eth1 VRF (10.1.0.0/24) │  10.1.0.11   │     │
│  └──────────────┘       BGP/OSPF/BFD      └──────────────┘     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Protocol Coverage

| Protocol | Default VRF (eth0) | VRF vrf-red (eth1) |
|----------|--------------------|--------------------|
| BGP      | 1 peer             | 1 peer             |
| OSPF     | Area 0             | Area 0             |
| BFD      | 1 peer             | 1 peer             |
| PIM      | Neighbor           | -                  |
| VRRP     | VRID 10            | -                  |
| Routes   | Static             | -                  |

## Requirements

VRF support requires a Linux kernel with `CONFIG_NET_VRF`. This module is not
available in Docker Desktop or Colima's lightweight VMs.

### Linux

Works out of the box as long as Docker is installed.

### macOS (Intel & Apple Silicon)

You need a full Ubuntu VM. The Makefile automates this via Lima:

```bash
# One-time setup
make lima-setup       # Create Ubuntu VM
make lima-provision   # Install Docker, make, and tools

# Shell into the VM (your mac filesystem is mounted)
make lima-shell

# Inside the VM: run the dev environment
make up build deploy
make metrics
```

**Lima VM management:**
```bash
make lima-stop     # Stop the VM
make lima-start    # Start it again
make lima-delete   # Remove completely
```

## Make Targets

```
FRR Exporter Development Environment
====================================

Quick Start:
  make up                              - Start FRR routers (FRR 10.5.1)
  make build                           - Build frr_exporter
  make deploy                          - Deploy exporter to containers
  make redeploy                        - Rebuild and redeploy exporter
  make down                            - Stop environment
  make restart                         - Restart environment
  make clean                           - Stop and remove built binary

Or all at once:
  make up build deploy                 - Start everything

Different FRR version:
  make up FRR_VERSION=7.5.1

Metrics:
  make metrics1                        - View metrics from router1
  make metrics2                        - View metrics from router2

Debugging:
  make vtysh1                          - Open vtysh on router1
  make vtysh2                          - Open vtysh on router2
  make status                          - Show all protocol status
  make show-bgp                        - Show BGP status
  make show-ospf                       - Show OSPF status
  make show-bfd                        - Show BFD status
  make show-pim                        - Show PIM status
  make show-vrrp                       - Show VRRP status
  make show-routes                     - Show route summary
  make show-version                    - Show FRR version
  make logs                            - View FRR container logs
  make exporter-logs                   - View exporter logs on router1 and router2
  make exporter-logs-follow-router1    - Tail exporter logs on router1
  make exporter-logs-follow-router2    - Tail exporter logs on router2

Testing different versions:
  make test-all-versions TEST_VERSIONS="8.5.7 10.5.1"

macOS Setup (VRF requires Lima VM):
  make lima-setup                      - Create Ubuntu VM
  make lima-provision                  - Install Docker and tools in VM
  make lima-shell                      - Shell into VM
  make lima-start                      - Start the VM
  make lima-stop                       - Stop the VM
  make lima-delete                     - Remove the VM

Endpoints (after deploy):
  http://localhost:9342/metrics        - Router1
  http://localhost:9343/metrics        - Router2
```

## Development Workflow

After making code changes:

```bash
make redeploy    # Rebuilds and restarts exporter
make metrics1
make metrics2
```

## Endpoints

| URL | Description |
|-----|-------------|
| http://localhost:9342/metrics | Router1 exporter |
| http://localhost:9343/metrics | Router2 exporter |

