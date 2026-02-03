# Configuration

## Environment Variables

### Required

| Variable {: style="width:35%" } | Description | Notes |
|---|---|---|
| `ADGUARD_HOME_HOST` | The URL of your AdGuard Home instance | Only required if `ADGUARD_HOME_DISABLED` is set to `false` |
| `ADGUARD_HOME_USERNAME` | Your AdGuard Home username | Only required if `ADGUARD_HOME_DISABLED` is set to `false` |
| `ADGUARD_HOME_PASSWORD` | Your AdGuard Home password | Only required if `ADGUARD_HOME_DISABLED` is set to `false` |
| `NGINX_PROXY_MANAGER_HOST` | The URL of your Nginx Proxy Manager instance. | |
| `NGINX_PROXY_MANAGER_USERNAME` | Your Nginx Proxy Manager username. | |
| `NGINX_PROXY_MANAGER_PASSWORD` | Your Nginx Proxy Manager password. <br> **Important:** It is recommended to create a new non-admin user with only the "Proxy Hosts - Manage" permission. | |
| `PIHOLE_HOST` | The URL of your Pi-Hole instance. | Only required if `PIHOLE_DISABLED` is set to `false` |
| `PIHOLE_PASSWORD` | Your Pi-Hole password. <br> **Important:** It is recommended to create an 'application password' rather than using your actual admin password. | Only required if `PIHOLE_DISABLED` is set to `false` |

### Optional

| Variable {: style="width:35%" } | Description | Default {: style="width:10%" } |
|---|---|---|
| `ADGUARD_HOME_DISABLED` | Set to `false` to enable AdGuard Home functionality | `true` |
| `DEBUG` | Set to `true` to enable DEBUG level logs | `false` |
| `DOCKER_HOST` | The URL of a docker socket proxy. If set, you don't need to mount the docker socket as a volume. Querying containers must be allowed (typically done by setting the `CONTAINERS` environment variable to `1`). | *None* |
| `DOCKER_HOSTS` | Comma-separated list of multiple docker hosts to monitor, with an empty string meaning the default local host.<br>For example `DOCKER_HOSTS=,tcp://192.168.0.101:2375` | `""` |
| `PIHOLE_DISABLED` | Set to `true` to disable Pi-Hole functionality | `false` |
| `RUN_INTERVAL` | The interval at which to scan for new containers, in Go's [`time.ParseDuration`](<https://go.dev/pkg/time/#ParseDuration>){: target="_blank" } format. Set to `0` to run once and exit. | `1h` |
| `TZ` | Customise the timezone. | *None* |

## Per Container Configuration

Use the following labels on your containers to enable specific features

### AdGuard Home

| Label {: style="width:45%"} | Description | Default {: style="width:10%"} |
|---|---|---|
| `plugNPiN.adguardHomeOptions.targetDomain` | If provided, a CNAME DNS Rewrite will be created  |  |

### Nginx Proxy Manager


| Label {: style="width:30%"} | Description | Default {: style="width:10%"} | Notes |
|---|---|---|---|
| `plugNPiN.npmOptions.advancedConfig` | Advanced nginx configuration (referred to as `Custom Nginx Configuration` in NPM UI) | | If using a docker compose file make sure to use `|` so new lines will be respected, for example:<pre><code>labels:<br>  - plugNPiN.ip=192.168.0.100:8000<br>  - plugNPiN.url=service.home<br>  - \|<br>    plugNPiN.npmOptions.advancedConfig=location / {<br>      allow 192.168.0.1/15;<br>      deny all;<br>    }</code></pre> |
| `plugNPiN.npmOptions.blockExploits` | Enables or disables the "Block Common Exploits" option on the proxy host. Set to `true` or `false` | `true` | |
| `plugNPiN.npmOptions.cachingEnabled` | Enables or disables the "Cache Assets" option on the proxy host. Set to `true` or `false`  | `false` | |
| `plugNPiN.npmOptions.certificateName` | Certificate to use for this host. Must already exist on the NPM instance |  | |
| `plugNPiN.npmOptions.forceSsl` | Force SSL | `false` | |
| `plugNPiN.npmOptions.http2Support` | Enable HTTP/2 Support | `false` | |
| `plugNPiN.npmOptions.hstsEnabled` | Enable HSTS | `false` | |
| `plugNPiN.npmOptions.hstsSubdomains` | Enable HSTS Subdomains | `false` | |
| `plugNPiN.npmOptions.scheme` | The scheme used to forward traffic to the container. Can be `http` or `https` | `http` | |
| `plugNPiN.npmOptions.websocketsSupport` | Enables or disables the "Allow Websocket Upgrade" option on the proxy host. Set to `true` or `false` | `false` | |

### Pi-Hole

| Label {: style="width:35%"} | Description | Default {: style="width:10%"} |
|---|---|---|
| `plugNPiN.piholeOptions.targetDomain` | If provided, a CNAME record will be created **instead** of a DNS record |  |

*[NPM]: Nginx Proxy Manager
