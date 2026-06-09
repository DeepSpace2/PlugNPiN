# Configuration

!!! tip

    Hover over any variable or label to reveal a 'copy to clipboard' button

## Docker Secrets

[:octicons-tag-24: 1.0.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v1.0.0){ .md-tag target="_blank" }


PlugNPiN supports [Docker Secrets](https://docs.docker.com/compose/how-tos/use-secrets/){: target="_blank" } for managing sensitive credentials and host URLs. This is the recommended approach for production deployments.

**Precedence:** Environment variables always take precedence over Docker Secrets.

**Usage:** Secrets' filenames must match the corresponding environment variable.

## Environment Variables

### Required

| Variable {: style="width:30%" } | Description | Notes |
|---|---|---|
| `ADGUARD_HOME_HOST`<br>[:octicons-tag-24: 0.8.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.8.0){ .md-tag target="_blank" } | The URL of your AdGuard Home instance | Only required if `ADGUARD_HOME_DISABLED` is set to `false`. Can be set using [Docker Secrets](#docker-secrets) |
| `ADGUARD_HOME_USERNAME`<br>[:octicons-tag-24: 0.8.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.8.0){ .md-tag target="_blank" } | Your AdGuard Home username | Only required if `ADGUARD_HOME_DISABLED` is set to `false`. Can be set using [Docker Secrets](#docker-secrets) |
| `ADGUARD_HOME_PASSWORD`<br>[:octicons-tag-24: 0.8.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.8.0){ .md-tag target="_blank" } | Your AdGuard Home password | Only required if `ADGUARD_HOME_DISABLED` is set to `false`. Can be set using [Docker Secrets](#docker-secrets) |
| `NGINX_PROXY_MANAGER_HOST`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | The URL of your Nginx Proxy Manager instance. | Can be set using [Docker Secrets](#docker-secrets) |
| `NGINX_PROXY_MANAGER_USERNAME`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | Your Nginx Proxy Manager username. | Can be set using [Docker Secrets](#docker-secrets) |
| `NGINX_PROXY_MANAGER_PASSWORD`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | Your Nginx Proxy Manager password. <br> **Important:** It is recommended to create a new non-admin user with only the "Proxy Hosts - Manage" permission. | Can be set using [Docker Secrets](#docker-secrets) |
| `PIHOLE_HOST`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | The URL of your Pi-Hole instance. | Only required if `PIHOLE_DISABLED` is set to `false`. Can be set using [Docker Secrets](#docker-secrets) |
| `PIHOLE_PASSWORD`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | Your Pi-Hole password. <br> **Important:** It is recommended to create an 'application password' rather than using your actual admin password. | Only required if `PIHOLE_DISABLED` is set to `false`. Can be set using [Docker Secrets](#docker-secrets) |

### Optional

| Variable {: style="width:30%" } | Description | Default {: style="width:10%" } |
|---|---|---|
| `ADGUARD_HOME_DISABLED`<br>[:octicons-tag-24: 0.8.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.8.0){ .md-tag target="_blank" } | Set to `false` to enable AdGuard Home functionality | `true` |
| `DEBUG`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | Set to `true` to enable DEBUG level logs | `false` |
| `DOCKER_HOSTS`<br>[:octicons-tag-24: 0.9.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.9.0){ .md-tag target="_blank" } | Comma-separated list of multiple docker hosts to monitor, with an empty string meaning the default local host.<br>For example `DOCKER_HOSTS=,tcp://192.168.0.101:2375` | `""` |
| `DOCKER_HOST`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | The URL of a docker socket proxy. If set, you don't need to mount the docker socket as a volume. Querying containers must be allowed (typically done by setting the `CONTAINERS` environment variable to `1`). | *None* |
| `METRICS`<br>[:octicons-tag-24: 1.0.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v1.0.0){ .md-tag target="_blank" } | Exposes a `/metrics` endpoint for Prometheus scraping. See [Monitoring → Prometheus](./monitoring.md#prometheus). | `false` |
| `METRICS_SERVER_PORT`<br>[:octicons-tag-24: 1.0.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v1.0.0){ .md-tag target="_blank" } | Port for the metrics endpoint. See [Monitoring → Prometheus](./monitoring.md#prometheus). | `9100` |
| `PIHOLE_DISABLED`<br>[:octicons-tag-24: 0.6.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.6.0){ .md-tag target="_blank" } | Set to `true` to disable Pi-Hole functionality | `false` |
| `RUN_INTERVAL`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | The interval at which to scan for new containers, in Go's [`time.ParseDuration`](<https://go.dev/pkg/time/#ParseDuration>){: target="_blank" } format. Set to `0` to run once and exit. | `1h` |
| `TZ`<br>[:octicons-tag-24: 0.1.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.1.0){ .md-tag target="_blank" } | Customise the timezone. | `""` |

## Per Container Configuration

Use the following labels on your containers to enable specific features

### General Options

| Label {: style="width:45%"} | Description | Default {: style="width:10%"} | Notes |
|---|---|---|---|
| `plugNPiN.options.createOnHealthy`<br>[:octicons-tag-24: 1.0.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v1.0.0){ .md-tag target="_blank" } | If set to `true`, PlugNPiN will wait for the container to become **healthy** before creating entries | `false` | **This option requires the container to have a [Docker Healthcheck](https://docs.docker.com/engine/reference/builder/#healthcheck){: target="_blank" } defined. If no healthcheck is found, an error will be logged and no entries will be created** |

### AdGuard Home

| Label {: style="width:45%"} | Description | Default {: style="width:10%"} | Notes |
|---|---|---|---|
| `plugNPiN.adguardHomeOptions.targetDomain`<br>[:octicons-tag-24: 0.8.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.8.0){ .md-tag target="_blank" } | If provided, a CNAME DNS Rewrite will be created | | |

### Nginx Proxy Manager


| Label {: style="width:35%"} | Description | Default {: style="width:10%"} | Notes |
|---|---|---|---|
| `plugNPiN.npmOptions.accessListName`<br>[:octicons-tag-24: 1.0.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v1.0.0){ .md-tag target="_blank" } | Access list to use for this host. Must already exist on the NPM instance | | |
| `plugNPiN.npmOptions.advancedConfig`<br>[:octicons-tag-24: 0.7.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.7.0){ .md-tag target="_blank" } | Advanced nginx configuration (referred to as `Custom Nginx Configuration` in NPM UI) | | If using a docker compose file make sure to use `|` so new lines will be respected, for example:<pre><code>labels:<br>  - plugNPiN.ip=192.168.0.100:8000<br>  - plugNPiN.url=service.home<br>  - \|<br>    plugNPiN.npmOptions.advancedConfig=location / {<br>      allow 192.168.0.1/15;<br>      deny all;<br>    }</code></pre> |
| `plugNPiN.npmOptions.blockExploits`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Enables or disables the "Block Common Exploits" option on the proxy host. Set to `true` or `false` | `true` | |
| `plugNPiN.npmOptions.cachingEnabled`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Enables or disables the "Cache Assets" option on the proxy host. Set to `true` or `false`  | `false` | |
| `plugNPiN.npmOptions.certificateName`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Certificate to use for this host. Must already exist on the NPM instance |  | |
| `plugNPiN.npmOptions.forceSsl`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Force SSL | `false` | |
| `plugNPiN.npmOptions.http2Support`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Enable HTTP/2 Support | `false` | |
| `plugNPiN.npmOptions.hstsEnabled`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Enable HSTS | `false` | |
| `plugNPiN.npmOptions.hstsSubdomains`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Enable HSTS Subdomains | `false` | |
| `plugNPiN.npmOptions.scheme`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | The scheme used to forward traffic to the container. Can be `http` or `https` | `http` | |
| `plugNPiN.npmOptions.websocketsSupport`<br>[:octicons-tag-24: 0.4.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.4.0){ .md-tag target="_blank" } | Enables or disables the "Allow Websocket Upgrade" option on the proxy host. Set to `true` or `false` | `false` | |

### Pi-Hole

| Label {: style="width:45%"} | Description | Default {: style="width:10%"} | Notes |
|---|---|---|---|
| `plugNPiN.piholeOptions.targetDomain`<br>[:octicons-tag-24: 0.5.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v0.5.0){ .md-tag target="_blank" } | If provided, a CNAME record will be created **instead** of a DNS record | | |

*[NPM]: Nginx Proxy Manager
