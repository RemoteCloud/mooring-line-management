# Configuration & Deployment Topology

All configuration is environment-driven (`api/internal/config/config.go`). The same
binary and schema serve both deployment scopes; scope is config, not code.

## Deployment scopes

`SCOPE=onboard` requires `VESSEL_ID` and rejects any request naming a different
vessel (scope-guard middleware). `SCOPE=shore` serves the whole fleet. Every row
carries `vessel_id` + an `origin` marker, so the same schema works for one vessel
(onboard) or all of them (shore). The two reconcile through asynchronous outbox/event
sync, never a live request path. See [architecture.md](architecture.md) §3.

## Local dev topologies (docker-compose, in `deploy/`)

The two stacks use distinct compose project names so they coexist on one host.

| | Onboard (`docker-compose.onboard.yml`) | Shore (`docker-compose.shore.yml`) |
|---|---|---|
| Scope | single vessel (`VESSEL_ID`) | fleet-wide |
| Web | `http://localhost:8090` | `http://localhost:8091` |
| API (host port) | `:8080` | `:8081` |
| Postgres (host port) | `5442` | `5433` |
| MinIO API / console | `9100` / `9101` | `9002` / `9003` |

Inside each stack the web container proxies `/api/*` to the API container (stripping
the `/api` prefix), matching the Vite dev proxy. The browser always talks to the web
origin; the API is reached under `/api`.

## Environment variables

### Core

| Var | Default | Notes |
|---|---|---|
| `SCOPE` | `shore` | `onboard` or `shore` |
| `VESSEL_ID` | — | required when `SCOPE=onboard` |
| `HTTP_ADDR` | `:8080` | API listen address |
| `DATABASE_URL` | `postgres://mooring:mooring@localhost:5432/mooring?sslmode=disable` | onboard local maps to host port `5442`, shore to `5433` |
| `AUTO_MIGRATE` | `true` | apply pending migrations at startup (`false` to disable) |

### Object storage (S3 / MinIO)

| Var | Default | Notes |
|---|---|---|
| `S3_ENDPOINT` | `http://localhost:9000` | internal endpoint used by the API |
| `S3_PUBLIC_ENDPOINT` | = `S3_ENDPOINT` | browser-reachable endpoint used to sign GET URLs (differs in Docker: internal host vs host-mapped port) |
| `S3_BUCKET` | `mooring` | bucket for certs, manuals, condition photos |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | `minioadmin` / `minioadmin` | credentials |
| `S3_REGION` | `us-east-1` | region |
| `S3_USE_SSL` | `false` | `true` to use TLS |

### Authentication (OIDC)

The OIDC / session / token-encryption variables are documented in full —
with meanings, examples and provider setup — in
[authentication.md](authentication.md#environment-variables).

## Common Make targets

```sh
make help          # list all targets
make build         # build the api binary -> bin/server
make test          # backend tests
make tidy          # go mod tidy

make onboard-up    # bring up onboard stack (db + minio + api + web)
make onboard-down
make shore-up      # bring up shore stack
make shore-down

make migrate-up    # apply migrations (uses DATABASE_URL or default)
make migrate-down  # roll back

make seed                # load Norwegian Luna demo data locally (RESET=1 to wipe first)
make seed-docker         # reseed inside the running onboard api container
make seed-docker-shore   # reseed inside the running shore api container

make openapi       # emit api/openapi/openapi.json (the OpenAPI 3.1 spec)
make gen-ts        # regenerate frontend TS types from the emitted spec
make run-onboard VESSEL_ID=<uuid>   # run api locally as onboard
make run-shore                      # run api locally as shore
```

## Run without compose (API only)

```sh
docker compose -f deploy/docker-compose.onboard.yml up -d db
export DATABASE_URL="postgres://mooring:mooring@localhost:5442/mooring?sslmode=disable"
make migrate-up
make run-shore           # or: make run-onboard VESSEL_ID=<uuid>
curl localhost:8080/health
```

Note: serving traffic still requires the OIDC env vars (see
[authentication.md](authentication.md)); `migrate`/`dump-openapi` subcommands do not.
