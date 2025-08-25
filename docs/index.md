[![Go Report Card](https://goreportcard.com/badge/github.com/DeepSpace2/PlugNPiN)](https://goreportcard.com/report/github.com/DeepSpace2/PlugNPiN)
![Build Status](https://github.com/DeepSpace2/PlugNPiN/actions/workflows/release.yml/badge.svg)
[![Release](https://img.shields.io/github/v/release/DeepSpace2/PlugNPiN)](https://github.com/DeepSpace2/PlugNPiN/releases)

# 🔌 PlugNPiN

**Plug and play your docker containers into Pi-Hole & Nginx Proxy Manager**

Automatically detect running Docker containers based on labels, add them
as local DNS records in **Pi-Hole** and create matching proxy hosts in
**Nginx Proxy Manager**.

## How It Works

PlugNPiN discovers services by scanning for Docker containers that have the following labels:

- `plugNPiN.ip` - The IP address and port of the container (e.g., `192.168.1.100:8080`).
- `plugNPiN.url` - The desired URL for the service (e.g., `my-service.local`).

The application operates in two complementary modes to keep your services synchronized:

1.  **Real-Time Event Listening**: The application actively listens for Docker container events. When a container with the required labels is **started**, **stopped**, or **killed**, the tool immediately adds or removes the corresponding DNS and proxy host entries. This ensures that your services are updated in real-time as containers change state.

2.  **Periodic Synchronization**: In addition to real-time events, the tool performs a full synchronization at a regular interval, defined by the `RUN_INTERVAL` environment variable. During this periodic run, it scans all running containers and ensures that their DNS and proxy configurations are correct. This acts as a self-healing mechanism, correcting any entries that might have been missed or become inconsistent.

When a container is processed in either mode, PlugNPiN will:

1.  Create a DNS record pointing the specified `url` to the `ip` address on **Pi-Hole**.
2.  Create a proxy host to route traffic from the `url` to the container's `ip` and `port` on **Nginx Proxy Manager**.

## Configuration

### Environment Variables

#### Required Environment Variables

- `NGINX_PROXY_MANAGER_HOST` - The URL of your Nginx Proxy Manager instance.
- `NGINX_PROXY_MANAGER_USERNAME` - Your Nginx Proxy Manager username.
- `NGINX_PROXY_MANAGER_PASSWORD` - Your Nginx Proxy Manager password.
- `PIHOLE_HOST` - The URL of your Pi-hole instance.
- `PIHOLE_PASSWORD` - Your Pi-hole password.

> [!IMPORTANT]
> **Pi-Hole** - It is recommended to create an 'application password' and set that as `PIHOLE_PASSWORD` rather than using your actual admin password

> [!IMPORTANT]
> **Nginx Proxy Manager** - It is recommended to create a new non-admin user with a single permission, "Proxy Hosts - Manage"

#### Optional Environment Variables

- `DOCKER_HOST` - The URL of a docker socket proxy. If set, you don't need to mount the docker socket as a volume.
- `RUN_INTERVAL` - The interval at which to scan for new containers. The default is `1h` (1 hour). Set to `0` to run once and exit.
- `TZ` - Customise the timezone.

### Flags

- `--dry-run`, `-d` - Simulates the process of adding DNS records and proxy hosts without making any actual changes to Pi-hole or Nginx Proxy Manager.

## Usage

### Docker Compose

Example compose files:

- Using a docker socket proxy (recommended)

```yaml
services:
  docker-proxy-test:
    image: lscr.io/linuxserver/socket-proxy:latest
    container_name: docker-proxy-test
    environment:
      - CONTAINERS=1
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    read_only: true
    tmpfs:
      - /run

  PlugNPiN:
    image: ghcr.io/deepspace2/plugnpin:latest
    depends_on:
      - docker-proxy-test
    container_name: PlugNPiN
    env_file:
      - .env
    restart: unless-stopped
```

- Not using a docker socket proxy (**NOT** recommended)

```yaml
services:
  PlugNPiN:
    image: ghcr.io/deepspace2/plugnpin:latest
    container_name: PlugNPiN
    env_file:
      - .env
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped
```

## Contributing

Contributions are very welcome! If you have a feature request, bug report, or want to contribute to the code, please feel free to open an issue or submit a pull request.
