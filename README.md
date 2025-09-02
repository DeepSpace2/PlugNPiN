[![Go Report Card](https://goreportcard.com/badge/github.com/DeepSpace2/PlugNPiN)](https://goreportcard.com/report/github.com/DeepSpace2/PlugNPiN)
![Build Status](https://github.com/DeepSpace2/PlugNPiN/actions/workflows/release.yml/badge.svg)
[![Release](https://img.shields.io/github/v/release/DeepSpace2/PlugNPiN)](https://github.com/DeepSpace2/PlugNPiN/releases)

# 🔌 PlugNPiN

**Plug and play your docker containers into Pi-Hole & Nginx Proxy Manager**

Automatically detect running Docker containers based on labels, add them
as local DNS records in **Pi-Hole** and create matching proxy hosts in
**Nginx Proxy Manager**.

## Key Features

- Automatic Docker container detection.
- Local DNS record creation in Pi-hole.
- Nginx Proxy Manager host creation.
- Support for Docker socket proxy.

**[See the documentation site for full setup and configuration.](https://deepspace2.github.io/PlugNPiN)**

## How it Works

PlugNPiN discovers services by scanning for Docker containers that have the following labels:

- `plugNPiN.ip` - The IP address and port of the container (e.g., `192.168.1.100:8080`).
- `plugNPiN.url` - The desired URL for the service (e.g., `my-service.local`).

The application operates in two complementary modes to keep your services synchronized:

1.  **Real-Time Event Listening**: The application actively listens for Docker container events. When a container with the required labels is **started**, **stopped**, or **killed**, the tool immediately adds or removes the corresponding DNS and proxy host entries. This ensures that your services are updated in real-time as containers change state.

2.  **Periodic Synchronization**: In addition to real-time events, the tool performs a full synchronization at a regular interval, defined by the `RUN_INTERVAL` environment variable. During this periodic run, it scans all running containers and ensures that their DNS and proxy configurations are correct. This acts as a self-healing mechanism, correcting any entries that might have been missed or become inconsistent.

When a container is processed in either mode, PlugNPiN will:

1.  Create a DNS record pointing the specified `url` to the `ip` address on **Pi-Hole**.
2.  Create a proxy host to route traffic from the `url` to the container's `ip` and `port` on **Nginx Proxy Manager**.

## Usage

### Docker Compose

It is **highly recommended** to use a Docker socket proxy to avoid giving the container direct access to the Docker daemon. This improves security by limiting the container's privileges.

#### Recommended: Using a Docker Socket Proxy

```yaml
services:
  socket-proxy:
    image: lscr.io/linuxserver/socket-proxy:latest
    container_name: socket-proxy
    environment:
      # Allow access to the container list
      - CONTAINERS=1
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    read_only: true
    tmpfs:
      - /run

  plugnpin:
    image: ghcr.io/deepspace2/plugnpin:latest
    container_name: plugnpin
    depends_on:
      - socket-proxy
    environment:
      - DOCKER_HOST=tcp://socket-proxy:2375
      - NGINX_PROXY_MANAGER_HOST=...
      - NGINX_PROXY_MANAGER_USERNAME=...
      - NGINX_PROXY_MANAGER_PASSWORD=...
      - PIHOLE_HOST=...
      - PIHOLE_PASSWORD=...
    restart: unless-stopped
```

#### Not Recommended: Mounting the Docker Socket

```yaml
services:
  plugnpin:
    image: ghcr.io/deepspace2/plugnpin:latest
    container_name: plugnpin
    environment:
      - NGINX_PROXY_MANAGER_HOST=...
      - NGINX_PROXY_MANAGER_USERNAME=...
      - NGINX_PROXY_MANAGER_PASSWORD=...
      - PIHOLE_HOST=...
      - PIHOLE_PASSWORD=...
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped
```

## Contributing

Contributions are very welcome! If you have a feature request, bug report, or want to contribute yourself, please feel free to open an issue or submit a pull request.
