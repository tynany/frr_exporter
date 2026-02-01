#!/bin/bash
# Startup script for FRR containers
# Creates VRF and configures interfaces before FRR starts

set -e

# Enable IP forwarding
sysctl -w net.ipv4.conf.all.forwarding=1 &>/dev/null || true

# Create VRF if kernel supports it
if ip link add vrf-red type vrf table 10 2>/dev/null; then
    echo "Created VRF vrf-red"
    ip link set vrf-red up

    # Move eth1 into VRF if it exists
    if ip link show eth1 &>/dev/null; then
        ip link set eth1 master vrf-red
        echo "Moved eth1 to VRF vrf-red"
    fi
else
    echo "VRF not supported by kernel - skipping VRF setup"
fi

# Start FRR
exec /usr/lib/frr/docker-start
