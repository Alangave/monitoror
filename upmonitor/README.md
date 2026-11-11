# upmonitor

A simple URL monitoring wall inspired by [Monitoror](https://monitoror.com): a single Go
binary embedding a Node.js-built UI, shipped as a rootless container.

## Features

- **UPTIME tiles**: periodic HTTP checks (URL, method, expected status codes, keyword,
  timeout, interval) with a Monitoror-like tile wall.
- **Uptime graphs**: each tile shows a bar graph of the last checks, the uptime
  percentage, the last response time and the duration since the last status change.
- **Live updates**: tile states are pushed to the UI over Server-Sent Events.
- **Single binary**: the UI is built with Node.js 24 + Vite and embedded in the Go
  binary (standard library only, zero runtime dependencies).
- **Rootless container**: distroless image running as a non-root user.

## Configuration

The configuration format is inspired by Monitoror's `config.json`:

```json
{
  "version": "1.0",
  "columns": 3,
  "tiles": [
    {
      "type": "UPTIME",
      "label": "My website",
      "params": {
        "url": "https://example.com",
        "method": "GET",
        "expectedStatus": [200],
        "keyword": "",
        "timeout": 2000,
        "interval": 30,
        "insecureTLS": false
      }
    }
  ]
}
```

| Param            | Default  | Description                                    |
| ---------------- | -------- | ---------------------------------------------- |
| `url`            | required | HTTP/HTTPS URL to check                        |
| `method`         | `GET`    | HTTP method                                    |
| `expectedStatus` | `[200]`  | Status codes considered as success             |
| `keyword`        | —        | String that must appear in the response body   |
| `timeout`        | `2000`   | Check timeout in milliseconds (max 30000)      |
| `interval`       | `30`     | Seconds between checks (min 5)                 |
| `insecureTLS`    | `false`  | Skip TLS certificate verification              |

The last 100 checks are kept in memory per tile; the graph shows the last 48.

## Usage

```bash
# From source
make build
UPMONITOR_CONFIG=config-example.json ./bin/upmonitor

# With Docker
docker build -t upmonitor .
docker run --rm -p 8080:8080 -v "$PWD/config-example.json:/config.json:ro" \
  -e UPMONITOR_CONFIG=/config.json upmonitor
```

Then open http://localhost:8080.

## Environment variables

| Variable           | Default        | Description                |
| ------------------ | -------------- | -------------------------- |
| `UPMONITOR_CONFIG` | `config.json`  | Path to the configuration  |
| `UPMONITOR_ADDR`   | `:8080`        | Listen address             |
| `UPMONITOR_PORT`   | `8080`         | Port (used if no address)  |

## API

- `GET /api/config` — wall configuration (columns + tile metadata)
- `GET /api/events` — SSE stream of tile state updates (`event: tile`)
- `GET /api/healthz` — liveness probe

## Development

```bash
make run        # run backend with the example config
cd web && npm run dev   # UI dev server with proxy to :8080
make test vet
```

## Requirements

- Go 1.27+ (backend)
- Node.js 24+ (UI build)
