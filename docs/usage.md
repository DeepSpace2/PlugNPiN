# Usage

## CLI Flags

| Flag {: style="width:35%" } | Description                                                                                                                            |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `--dry-run`, `-d`           | Simulates the process of adding DNS/CNAME records and proxy hosts without making any actual changes to Pi-Hole or Nginx Proxy Manager. |

## Docker Compose

It is **highly recommended** to use a Docker socket proxy to avoid giving the container direct access to the Docker daemon. This improves security by limiting the container's privileges.

=== "Recommended: Using a Docker Socket Proxy"

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

=== "Not Recommended: Mounting the Docker Socket"

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
          - PIHOLE_PASSWORD...
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock:ro
        restart: unless-stopped
    ```
