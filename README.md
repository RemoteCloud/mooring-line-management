# Mooring Line Management System

Fleet-wide mooring line management for Norwegian Cruise Line. Tracks mooring lines,
their components, inspections, turning/side-tracking, certificates and condition over
time across a fleet of vessels.

One codebase, two deployments distinguished only by configuration:

- **Onboard** (per vessel) — own API + own Postgres, scoped to a single vessel. Works
  with no shore connectivity.
- **Shore** (global) — own API + own Postgres, fleet-wide consolidation and reporting.

The two reconcile through asynchronous outbox/event sync, never a live request path.

See [PLAN.md](PLAN.md) for architecture and build sequencing.

## Stack

- **Backend:** Go + [Huma v2](https://huma.rocks) (code-first, emits OpenAPI **3.1**),
  PostgreSQL (pgx), golang-migrate (embedded), S3-compatible object storage.
- **Frontend:** React + TypeScript + Vite + PWA (added in a later slice). Types are
  generated from the emitted OpenAPI spec via `openapi-typescript`.

> Why code-first: spec-first Go codegen (oapi-codegen, ogen) cannot yet parse OpenAPI
> 3.1 nullability. Huma emits a valid 3.1 spec from Go, which `openapi-typescript`
> consumes cleanly — keeping the 3.1 requirement and shared types both sides.

## Layout

```
api/                 Go backend
  cmd/server/        entrypoint: serve | migrate | dump-openapi
  internal/          config, store, httpapi (Huma), dbmigrate
  db/migrations/     golang-migrate SQL (embedded in the binary)
  openapi/           openapi.json — emitted 3.1 spec (build artifact)
deploy/              docker-compose (onboard / shore) + Dockerfile
web/                 React PWA (later slice)
Makefile             dev tasks
```

## Prerequisites

- Go 1.26+
- Docker (for Postgres + MinIO)
- Node 20+ (frontend slice)

## Quick start (local, via Docker)

Bring up the onboard stack (Postgres + MinIO + API):

```sh
make onboard-up          # builds + starts db, minio, api on :8080
```

Or run pieces by hand. Start just the database, then migrate and run the API:

```sh
docker compose -f deploy/docker-compose.onboard.yml up -d db
export DATABASE_URL="postgres://mooring:mooring@localhost:5442/mooring?sslmode=disable"
make migrate-up
make run-shore           # or: make run-onboard VESSEL_ID=<uuid>
curl localhost:8080/health
```

Shore stack (separate ports, fleet scope):

```sh
make shore-up            # api on :8081, db on :5433, minio on :9002/:9003
```

## Configuration (environment variables)

| Var | Default | Notes |
|---|---|---|
| `SCOPE` | `shore` | `onboard` or `shore` |
| `VESSEL_ID` | — | required when `SCOPE=onboard` |
| `HTTP_ADDR` | `:8080` | listen address |
| `DATABASE_URL` | `postgres://mooring:mooring@localhost:5432/mooring?sslmode=disable` | onboard local maps to host port `5442` |
| `S3_ENDPOINT` / `S3_BUCKET` / `S3_ACCESS_KEY` / `S3_SECRET_KEY` / `S3_REGION` / `S3_USE_SSL` | MinIO dev defaults | object storage |
| `JWT_SECRET` | `dev-insecure-change-me` | set in production |

## Common tasks

```sh
make build         # build api binary -> bin/server
make test          # backend tests
make migrate-up    # apply migrations
make migrate-down  # roll back
make openapi       # emit api/openapi/openapi.json (OpenAPI 3.1)
make gen-ts        # regenerate frontend TS types from the spec
make help          # list all targets
```

## Deployment scopes

`SCOPE=onboard` requires `VESSEL_ID` and rejects any request naming a different
vessel (scope guard middleware). `SCOPE=shore` serves the whole fleet. The same
binary and schema serve both — every row carries `vessel_id` + an `origin` marker.

## Status

- **Step 0 — foundations:** config, Huma API (OpenAPI 3.1), embedded migrations, scope
  guard, health, onboard/shore compose.
- **Slice F0 — frontend shell:** PWA, typed client from the spec, scope-aware nav,
  responsive/tablet, vessel switcher.
- **Slice 1 — deck map + rope register:** catalogue + vessel/layout + lines API,
  Norwegian Luna seed (`make seed`), deck map (winches/drums/rotation/edit-layout) and
  rope register (filter/sort/search, add-line, rope record with 4 tabs).

After `make onboard-up`, load demo data with `make seed-docker`, then open
**http://localhost:8090**.

Remaining slices (turning, inspections, files, dashboard, webhooks, sync) follow per
[PLAN.md](PLAN.md) §5.

> Implementation note: the query layer is hand-written pgx (not sqlc as originally
> planned) — the dynamic line-list filters and layout aggregates are clearer as
> explicit SQL, and it avoids adding the sqlc codegen tool. Same intent: typed,
> migrations-based, no ad-hoc schema.
