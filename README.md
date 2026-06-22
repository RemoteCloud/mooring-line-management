# Mooring Line Management System

Fleet-wide mooring line lifecycle management for Norwegian Cruise Line. Tracks
mooring lines and their components, inspections, turning/side-tracking, certificates,
photos and condition over time — with a deck map, rope register, dashboard and
condition reports. It runs in two scopes from one codebase: **onboard** (a single
vessel, works offline) and **shore** (fleet-wide consolidation). Access is gated by
**OIDC single sign-on**; the `admin` group gets read+write, everyone else is
read-only.

- **Backend:** Go 1.26 + [Huma v2](https://huma.rocks) (code-first, emits OpenAPI
  3.1), PostgreSQL (pgx), golang-migrate, S3/MinIO object storage.
- **Frontend:** React 19 + TypeScript + Vite (PWA), types generated from the spec.

## Quick start

You need Docker, plus an OIDC client (id + secret) registered with the Maranics
nightly provider — see [docs/authentication.md](docs/authentication.md) for how to
get one and register the redirect URI.

```sh
cp .env.example .env
# Edit .env: set OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, and a TOKEN_ENC_KEY
#   (generate one with: openssl rand -base64 32)

make shore-up          # web :8091, api :8081, db :5433, minio :9002/:9003
make seed-docker-shore # load Norwegian Luna demo data
```

Open **http://localhost:8091** and sign in. (For a single-vessel deployment use
`make onboard-up` + `make seed-docker`, then open **http://localhost:8090**.)

The default local redirect URI is `http://localhost:8091/api/auth/callback` — it must
be registered with the provider and match `OIDC_REDIRECT_URI`.

## Documentation

- [docs/architecture.md](docs/architecture.md) — system design, onboard/shore split,
  sync model, data model, build sequence.
- [docs/authentication.md](docs/authentication.md) — OIDC BFF flow, auth env vars,
  group-based permissions, redirect-URI registration, troubleshooting.
- [docs/configuration.md](docs/configuration.md) — all environment variables, dev
  topology/ports, and Make targets.
- [docs/](docs/) — documentation index.

## Layout

```
api/        Go backend (cmd/server, internal/{config,store,httpapi,auth,...}, db/migrations, openapi)
web/        React PWA (src/app, src/features, src/api generated types)
deploy/     docker-compose (onboard / shore) + Dockerfiles
docs/       architecture, authentication, configuration
Makefile    dev tasks (run `make help`)
```

## Status

Implemented: foundations (Huma API, OpenAPI 3.1, embedded migrations, scope guard,
health), frontend shell (PWA, typed client, scope-aware nav, vessel switcher),
deck map + rope register, turning/side tracking, inspections, files, dashboard,
catalogue, and **OIDC authentication + group-based permissions**. Remaining slices
(webhooks, onboard↔shore sync) follow [docs/architecture.md](docs/architecture.md) §5.
