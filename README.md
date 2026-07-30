# go-gateway

An HTTP API gateway written in Go. It sits in front of your backend services and handles request validation, rate limiting, routing, proxying, CORS, and request auditing — so each service does not have to reinvent those concerns.

Requests are matched to a backend by `service_name` in the path, then forwarded to the configured `service_url` after optional redirects, auth checks, and rate limits.

---

## How it works

Every request moves through an explicit lifecycle. Progress is modeled as a **state machine**: only valid transitions are allowed, and failures are first-class states (not ad-hoc early returns). That makes the pipeline easy to reason about and straightforward to audit end-to-end.

### Request pipeline

```text
  Client
    │
    ▼
┌──────────┐   ┌──────────┐   ┌────────────┐   ┌─────────┐   ┌─────────┐
│   Map    │──▶│ Validate │──▶│ Rate limit │──▶│  Route  │──▶│  Proxy  │
│ service  │   │  policy  │   │            │   │         │   │upstream │
└──────────┘   └──────────┘   └────────────┘   └─────────┘   └────┬────┘
                                                                  │
                                                                  ▼
                                                              Response
```

| Stage | What it does |
|-------|----------------|
| **Map** | Resolve which configured backend the path belongs to |
| **Validate** | Apply edge policy (IPs, origins, headers, bearer presence, deprecation/version rules) |
| **Rate limit** | Enforce global / service / route limits |
| **Route** | Optionally rewrite the path from config-driven match rules |
| **Proxy** | Forward to the upstream and return the response |

Timeouts can fire while a request is in flight. Stage failures and timeouts are recorded so you can see *where* the request stopped.

### Request state machine

Happy path moves left → right. Each stage can fail; those failures converge on a terminal **Failed** state. **Timeout** can occur from any in-flight stage.

```mermaid
stateDiagram-v2
    [*] --> Initialized

    Initialized --> Mapped: map ok
    Initialized --> MapFailed: map failed

    Mapped --> Validated: validate ok
    Mapped --> ValidationFailed: validate failed

    Validated --> RateLimited: limit ok
    Validated --> RateLimitFailed: limit exceeded

    RateLimited --> Proxied: proxy ok
    RateLimited --> ProxyFailed: proxy failed

    Proxied --> Success: respond
    Success --> [*]

    MapFailed --> Failed
    ValidationFailed --> Failed
    RateLimitFailed --> Failed
    ProxyFailed --> Failed
    Failed --> [*]

    Initialized --> TimedOut: timeout
    Mapped --> TimedOut: timeout
    Validated --> TimedOut: timeout
    RateLimited --> TimedOut: timeout
    Proxied --> TimedOut: timeout
    TimedOut --> [*]
```

In short: the gateway does not only forward HTTP — it advances each request through a constrained set of states so illegal progress is hard and outcomes stay visible.

---

## Prerequisites

- [Go](https://go.dev/dl/) **1.25+**
- [PostgreSQL](https://www.postgresql.org/) (used to persist request logs)
- Optional: [Docker](https://www.docker.com/) if you want to run Postgres via Compose

---

## Quick start

### 1. Install dependencies

From the project root:

```bash
go mod download
```

### 2. Environment

Create a `.env` file in the **project root**. The gateway loads it on startup via `godotenv`.

| Variable        | Required | Description                                      |
|-----------------|----------|--------------------------------------------------|
| `DATABASE_URL`  | Yes      | PostgreSQL connection string used for audit logs |

Example:

```env
DATABASE_URL=postgresql://admin:admin@localhost:5432/mydb
```

### 3. Database

A Postgres instance must be reachable before the gateway starts. You can bring one up with the included Compose file:

```bash
docker compose up -d
```

That starts Postgres on `5432` with user `admin`, password `admin`, and database `mydb` (matching the example `DATABASE_URL` above).

The gateway writes to a `request_logs` table. Create it if it does not already exist:

```sql
CREATE TABLE IF NOT EXISTS request_logs (
  req_id     TEXT,
  created_at TIMESTAMPTZ,
  source     TEXT,
  data       JSONB
);
```

### 4. Configuration

Place a `config.json` file in the **project root**. The gateway loads `./config.json` on startup. A sample config is already present — use it as a starting point and point `service_url` values at your backends.

### 5. Run

```bash
go run .
```

The gateway starts an HTTP server and begins accepting traffic. Graceful shutdown is handled on `SIGINT` / `SIGTERM`.

Request logs can be inspected at `GET /logs`.

---

## Configuration reference (`config.json`)

Top-level and per-service options are defined and validated in [`internal/config/config.go`](internal/config/config.go). Below is a summary of the fields.

### Global options

| Field                   | JSON key               | Description |
|-------------------------|------------------------|-------------|
| `Port`                  | `port`                 | Port the gateway listens on |
| `GlobalTimeoutInSeconds`| `global_timeout_sec`   | Max time for reading headers/body and writing the response. Also used as the default proxy timeout. The global request context timeout is **2×** this value |
| `RateLimitPerMinute`    | `rate_limit_per_min`   | Global rate limit (requests per minute) |
| `ReqSizeLimitInBytes`   | `req_size_in_bytes`    | Maximum allowed request body size |
| `BlockedIps`            | `blocked_ips`          | Globally blocked client IPs |
| `AllowedOrigins`        | `allowed_origins`      | Globally permitted CORS origins |
| `HealthCheckIntervalInSeconds` | `health_interval_sec` | Interval between backend health checks |
| `Services`              | `services`             | List of backend service configs |

### Service options

| Field          | JSON key       | Description |
|----------------|----------------|-------------|
| `ServiceName`  | `service_name` | Identifier used to match the service from the request path |
| `ServiceUrl`   | `service_url`  | Upstream base URL where matching requests are proxied |
| `TimeoutInSeconds` | `timeout_sec` | Service-level request timeout (optional) |
| `RateLimitOpts` | `rate_limit_opts` | Service / route rate limits |
| `RoutingOpts`  | `redirect_opts` | Conditional path redirects |
| `ReqProxyOpts` | `proxy_opts`   | Proxy timeout overrides |
| `AuthOpts`     | `auth_opts`    | Validation, policy, deprecation, and versioning |

#### Rate limits (`rate_limit_opts`)

- `rate_limit_per_min` — service-level limit
- `route_level_limit_per_min` — map of path → limit

#### Routing (`redirect_opts` → `req_route_config`)

Per-path rules that redirect when conditions match:

- `headers` / `headers_regex`
- `host` / `host_regex`
- `methods`
- `query` / `query_regex`
- `client_ip`
- `redirect_path` — target path when a rule matches

#### Proxy (`proxy_opts`)

- `proxy_req_timeout_sec` — timeout while proxying to the upstream

#### Auth / policy (`auth_opts`)

- **`validator_opts`** — `required_headers`, `restricted_headers`, `allowed_headers`
- **`deprecation_opts`** — `deprecated_urls`, `deprecated_headers`, `obsolete_urls`
- **`policy_opts`**
  - `blocked_ip_opts` — service / route blocked IPs
  - `req_origin_opts` — service-level allowed origins
  - `bearer_token_policy_opts` — require a bearer token, with optional skip paths
- **`version_opts`** — `available_versions` (e.g. `v1`), `default_version`

For exact struct tags and validation rules, see [`internal/config/config.go`](internal/config/config.go).

---

## How requests are matched

The gateway looks for a configured `service_name` in the request path. For example, with `service_name: "orders"`, a request to `/orders/checkout` is mapped to that service and proxied to its `service_url`.

If no configured service name appears in the path, the request is rejected.

---

## Tests

Run the full suite from the project root:

```bash
go test ./...
```

Unit tests live next to the packages under `internal/`. End-to-end coverage is under `tests/e2e/` and expects a working `.env` (`DATABASE_URL`) and `config.json`, plus a reachable Postgres instance.

To run a single package:

```bash
go test ./internal/validate/ -v
```

---

## Project layout

```
.
├── main.go                 # Entry point — wires config, store, pipeline, HTTP server
├── config.json             # Runtime gateway configuration (project root)
├── docker-compose.yaml     # Local Postgres
├── internal/
│   ├── auditor/            # Request audit trail
│   ├── config/             # Config loading & validation
│   ├── gateway/            # Startup / initialization
│   ├── proxy/              # Upstream proxying
│   ├── rate-limit/         # Token-bucket rate limiting
│   ├── request/            # Request pipeline
│   ├── route/              # Conditional routing
│   ├── services/           # Service path mapping
│   ├── store/              # Postgres persistence
│   ├── validate/           # Request validation & policies
│   └── ...
├── tests/e2e/              # End-to-end tests
└── docs/                   # Design notes (routing, rate limiting, logging, etc.)
```

Design write-ups under [`docs/`](docs/) cover rate limiting, routing, validation, persistence, and logging in more depth if you want the “why” behind the pipeline.
