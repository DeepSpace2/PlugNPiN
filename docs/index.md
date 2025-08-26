[![Go Report Card](https://goreportcard.com/badge/github.com/DeepSpace2/PlugNPiN)](https://goreportcard.com/report/github.com/DeepSpace2/PlugNPiN)
![Build Status](https://github.com/DeepSpace2/PlugNPiN/actions/workflows/release.yml/badge.svg)
[![Release](https://img.shields.io/github/v/release/DeepSpace2/PlugNPiN)](https://github.com/DeepSpace2/PlugNPiN/releases)

# 🔌 PlugNPiN

**Plug and play your docker containers into Pi-Hole & Nginx Proxy Manager**

Automatically detect running Docker containers based on labels, add them
as local DNS records in **Pi-Hole** and create matching proxy hosts in
**Nginx Proxy Manager**.

## How it Works

PlugNPiN scans for Docker containers with the following labels:

- `plugNPiN.ip` - The IP address and port of the container (e.g., `192.168.1.100:8080`).
- `plugNPiN.url` - The desired URL for the service (e.g., `my-service.local`).

When a container with these labels is found, PlugNPiN will:

1. Create a DNS record pointing the specified `url` to the `ip` address on **Pi-Hole**.
2. Create a proxy host to route traffic from the `url` to the container's `ip` and `port` on **Nginx Proxy Manager**.

## Configuration

### Environment Variables

#### Required

| Variable {: style="width:35%" } | Description |
|---|---|
| `NGINX_PROXY_MANAGER_HOST` | The URL of your Nginx Proxy Manager instance. |
| `NGINX_PROXY_MANAGER_USERNAME` | Your Nginx Proxy Manager username. |
| `NGINX_PROXY_MANAGER_PASSWORD` | Your Nginx Proxy Manager password. <br> **Important:** It is recommended to create a new non-admin user with only the "Proxy Hosts - Manage" permission. |
| `PIHOLE_HOST` | The URL of your Pi-hole instance. |
| `PIHOLE_PASSWORD` | Your Pi-hole password. <br> **Important:** It is recommended to create an 'application password' rather than using your actual admin password. |

#### Optional

| Variable {: style="width:35%" } | Description | Default {: style="width:10%" } |
|---|---|---|
| `DOCKER_HOST` | The URL of a docker socket proxy. If set, you don't need to mount the docker socket as a volume. Querying containers must be allowed (typically done by setting the `CONTAINERS` environment variable to `1`). | *None* |
| `RUN_INTERVAL` | The interval at which to scan for new containers, in Go's [`time.ParseDuration`](https://go.dev/pkg/time/#ParseDuration) format. Set to `0` to run once and exit. | `1h` |
| `TZ` | Customise the timezone. | *None* |

### Flags

| Flag {: style="width:35%" } | Description |
|---|---|
| `--dry-run`, `-d` | Simulates the process of adding DNS records and proxy hosts without making any actual changes to Pi-hole or Nginx Proxy Manager. |

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

