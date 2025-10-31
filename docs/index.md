[![Go Report Card](https://goreportcard.com/badge/github.com/DeepSpace2/PlugNPiN)](https://goreportcard.com/report/github.com/DeepSpace2/PlugNPiN)
![Build Status](https://github.com/DeepSpace2/PlugNPiN/actions/workflows/release.yml/badge.svg)
[![Release](https://img.shields.io/github/v/release/DeepSpace2/PlugNPiN)](https://github.com/DeepSpace2/PlugNPiN/releases)

# 🔌 PlugNPiN

**Plug and play your docker containers into Pi-Hole & Nginx Proxy Manager**

Automatically detect running Docker containers based on labels, add them
as local DNS/[CNAME](#targetDomainLabel) records in **Pi-Hole** and create matching proxy hosts in
**Nginx Proxy Manager**.

## How It Works

PlugNPiN discovers services by scanning for Docker containers that have the following labels:

- `plugNPiN.ip` - The IP address and port of the container (e.g., `192.168.1.100:8080`).
- `plugNPiN.url` - The desired URL for the service (e.g., `my-service.local`).

The application operates in two complementary modes to keep your services synchronized:

1. **Real-Time Event Listening**: The application actively listens for Docker container events. When a container with the required labels is **started**, **stopped**, or **killed**, the tool immediately adds or removes the corresponding DNS and proxy host entries. This ensures that your services are updated in real-time as containers change state.

2. **Periodic Synchronization**: In addition to real-time events, the tool performs a full synchronization at a regular interval, defined by the `RUN_INTERVAL` environment variable. During this periodic run, it scans all running containers and ensures that their DNS and proxy configurations are correct. This acts as a self-healing mechanism, correcting any entries that might have been missed or become inconsistent.

When a container is processed in either mode, PlugNPiN will:

1. Create a DNS record pointing the specified `url` to the `ip` address on **Pi-Hole** (or a [CNAME record](#targetDomainLabel) pointing to a configurable target domain).
2. Create a proxy host to route traffic from the `url` to the container's `ip` and `port` on **Nginx Proxy Manager**.

### CNAME Records

It is possible to force PlugNPiN to create CNAME records instead of local DNS records ("A record") in Pi-Hole by setting the `plugNPiN.piholeOptions.targetDomain` label.
See [Per Container Configuration ➔ Pi-Hole](#targetDomainLabel).

## Configuration

### Environment Variables

#### Required

| Variable {: style="width:35%" } | Description | Notes |
|---|---|---|
| `NGINX_PROXY_MANAGER_HOST` | The URL of your Nginx Proxy Manager instance. | |
| `NGINX_PROXY_MANAGER_USERNAME` | Your Nginx Proxy Manager username. | |
| `NGINX_PROXY_MANAGER_PASSWORD` | Your Nginx Proxy Manager password. <br> **Important:** It is recommended to create a new non-admin user with only the "Proxy Hosts - Manage" permission. | |
| `PIHOLE_HOST` | The URL of your Pi-Hole instance. | Only required if [`PIHOLE_DISABLED`](#piHoleDisabledEnvVar) is set to `false` |
| `PIHOLE_PASSWORD` | Your Pi-Hole password. <br> **Important:** It is recommended to create an 'application password' rather than using your actual admin password. | Only required if [`PIHOLE_DISABLED`](#piHoleDisabledEnvVar) is set to `false` |

#### Optional

| Variable {: style="width:35%" } | Description | Default {: style="width:10%" } |
|---|---|---|
| `DOCKER_HOST` | The URL of a docker socket proxy. If set, you don't need to mount the docker socket as a volume. Querying containers must be allowed (typically done by setting the `CONTAINERS` environment variable to `1`). | *None* |
| <div id="piHoleDisabledEnvVar"><a name="piholeDisabledEnvVar"></a>`PIHOLE_DISABLED`</div> | Set to `true` to disable Pi-Hole functionality | `false` |
| `RUN_INTERVAL` | The interval at which to scan for new containers, in Go's [`time.ParseDuration`](<https://go.dev/pkg/time/#ParseDuration>){: target="_blank" } format. Set to `0` to run once and exit. | `1h` |
| `TZ` | Customise the timezone. | *None* |

### Flags

| Flag {: style="width:35%" } | Description |
|---|---|
| `--dry-run`, `-d` | Simulates the process of adding DNS/CNAME records and proxy hosts without making any actual changes to Pi-Hole or Nginx Proxy Manager. |

### Per Container Configuration

#### Nginx Proxy Manager

Use the following labels to configure Nginx Proxy Manager entries

| Label {: style="width:35%"} | Description | Default {: style="width:10%"} |
|---|---|---|
| `plugNPiN.npmOptions.blockExploits` | Enables or disables the "Block Common Exploits" option on the proxy host. Set to `true` or `false` | `true` |
| `plugNPiN.npmOptions.cachingEnabled` | Enables or disables the "Cache Assets" option on the proxy host. Set to `true` or `false`  | `false` |
| `plugNPiN.npmOptions.certificateName` | Certificate to use for this host. Must already exist on the NPM instance |  |
| `plugNPiN.npmOptions.forceSsl` | Force SSL | `false` |
| `plugNPiN.npmOptions.http2Support` | Enable HTTP/2 Support | `false` |
| `plugNPiN.npmOptions.hstsEnabled` | Enable HSTS | `false` |
| `plugNPiN.npmOptions.hstsSubdomains` | Enable HSTS Subdomains | `false` |
| `plugNPiN.npmOptions.scheme` | The scheme used to forward traffic to the container. Can be `http` or `https` | `http` |
| `plugNPiN.npmOptions.websocketsSupport` | Enables or disables the "Allow Websocket Upgrade" option on the proxy host. Set to `true` or `false` | `false` |

#### Pi-Hole

| Label {: style="width:35%"} | Description | Default {: style="width:10%"} |
|---|---|---|
| <div id="targetDomainLabel"><a name="targetDomainLabel"></a>`plugNPiN.piholeOptions.targetDomain`</div> | If provided, a CNAME record will be created **instead** of a DNS record |  |

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
