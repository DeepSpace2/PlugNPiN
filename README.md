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

**[See the documentation site for full setup and configuration.](https://deepspace2.github.io/plugnpin)**

## How it Works

PlugNPiN scans for Docker containers with the following labels:

- `plugNPiN.ip` - The IP address and port of the container (e.g., `192.168.1.100:8080`).
- `plugNPiN.url` - The desired URL for the service (e.g., `my-service.local`).

When a container with these labels is found, PlugNPiN will:

1. Create a DNS record pointing the specified `url` to the `ip` address on **Pi-Hole**.
2. Create a proxy host to route traffic from the `url` to the container's `ip` and `port` on **Nginx Proxy Manager**.

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
