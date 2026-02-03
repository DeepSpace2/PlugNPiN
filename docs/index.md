[![Go Report Card](https://goreportcard.com/badge/github.com/DeepSpace2/PlugNPiN)](https://goreportcard.com/report/github.com/DeepSpace2/PlugNPiN)
![Build Status](https://github.com/DeepSpace2/PlugNPiN/actions/workflows/release.yml/badge.svg)
[![Release](https://img.shields.io/github/v/release/DeepSpace2/PlugNPiN)](https://github.com/DeepSpace2/PlugNPiN/releases)
![Pulls](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fghcr-badge.elias.eu.org%2Fapi%2Fdeepspace2%2Fdeepspace2%2Fplugnpin&query=downloadCount&style=flat&logo=docker&label=Docker%20Pulls&color=2496ed)

# 🔌 PlugNPiN

**Plug and play your docker containers into Pi-Hole/AdGuard Home & Nginx Proxy Manager**

Automatically detect running Docker containers based on labels, add them
as local DNS/CNAME records in **Pi-Hole** (or DNS Rewrites in **AdGuard Home**) and create matching proxy hosts in
**Nginx Proxy Manager**.

**Pi-Hole's and AdGuard Home's functionality can be toggled individually. By default Pi-Hole is enabled and AdGuard Home is disabled.**
See [Optional Environment Variables](./configuration.md#optional).


## How It Works

PlugNPiN discovers services by scanning for Docker containers that have the following labels:

- `plugNPiN.ip` - The IP address and port of the container (e.g., `192.168.1.100:8080`).
- `plugNPiN.url` - The desired URL for the service (e.g., `my-service.local`).

The application operates in two complementary modes to keep your services synchronized:

1. **Real-Time Event Listening**: The application actively listens for Docker container events. When a container with the required labels is **started**, **stopped**, or **killed**, the tool immediately adds or removes the corresponding DNS and proxy host entries. This ensures that your services are updated in real-time as containers change state.

2. **Periodic Synchronization**: In addition to real-time events, the tool performs a full synchronization at a regular interval, defined by the `RUN_INTERVAL` environment variable. During this periodic run, it scans all running containers and ensures that their DNS and proxy configurations are correct. This acts as a self-healing mechanism, correcting any entries that might have been missed or become inconsistent.

When a container is processed in either mode, PlugNPiN will:

1. Create a DNS record pointing the specified `url` to the `ip` address on **Pi-Hole/AdGuard Home** (or a CNAME record pointing to a configurable target domain).
2. Create a proxy host to route traffic from the `url` to the container's `ip` and `port` on **Nginx Proxy Manager**.

### CNAME Records

#### AdGuard Home

To create a DNS Rewrite as a CNAME, set the `plugNPiN.adguardHomeOptions.targetDomain` label.

See [Per Container Configuration ➔ AdGuard Home](./configuration.md#adguard-home).

#### Pi-Hole

To create A CNAME record instead of local DNS records ("A record"), set the `plugNPiN.piholeOptions.targetDomain` label.

See [Per Container Configuration ➔ Pi-Hole](./configuration.md#pi-hole).
