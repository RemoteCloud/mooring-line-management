# Mooring Line Management — Architecture & Implementation Plan

**Stack:** Go backend (code-first OpenAPI 3.1) · PostgreSQL · React/TS PWA · S3-compatible object storage.
**Deployments:** one binary, two scopes (onboard single-vessel / shore fleet), set by config. Async outbox sync between them.

> This document is the original design/build-sequencing plan, kept as the canonical
> architecture reference. A few decisions have since shifted in implementation —
> notably **auth is now OIDC (not JWT/RBAC)** and **queries are hand-written pgx
> (not sqlc)**. Those points are annotated inline below. For the auth design as
> built, see [authentication.md](authentication.md).

---

## 1. Stack & key libraries

| Concern | Choice | Why |
|---|---|---|
| API framework + contract | **Huma v2** (code-first) | Emits **OpenAPI 3.1.0** natively from Go handlers+structs; built-in validation from struct tags. **De-risk (2026-06-07): spec-first via oapi-codegen AND ogen both fail on 3.1 `type:[x,"null"]` nullability** — they model `type` as scalar string. Code-first goes Go→3.1 (not 3.1→Go), sidestepping the parser. Verified: Huma emits 3.1.0, openapi-typescript consumes it cleanly, nullability flows via `*T` pointers. |
| Router | Huma `humago` adapter over stdlib `net/http` mux | Huma is router-agnostic; stdlib mux keeps deps light (chi available if richer routing needed) |
| DB queries | hand-written `pgx` *(plan said `sqlc`)* | The dynamic line-list filters and layout aggregates read clearer as explicit SQL; same intent (typed, migrations-based, no ad-hoc schema) without the extra codegen tool. |
| Migrations | `golang-migrate` | Versioned `.sql` up/down; no ad-hoc schema edits |
| Validation | generated from OpenAPI + domain guards | Request shape from spec; business rules in domain layer |
| Auth | OIDC Backend-for-Frontend + group-based authz *(plan said JWT + RBAC)* | External IdP (Maranics UserManagement); tokens stay server-side, browser holds an opaque session cookie. `admin` group = read+write, any other authenticated user = read-only. See [authentication.md](authentication.md). |
| Object storage | `aws-sdk-go-v2` (S3) / MinIO local | Certs, manuals, photos; DB holds refs only |
| Webhooks | outbox + dispatcher worker, HMAC-SHA256 | Signed, retryable |
| Sync | outbox/event replication worker | Append-only ops; tolerant of long offline gaps |
| Frontend client | `openapi-typescript` from Huma-emitted 3.1 spec | Shared types front↔back: Go structs → 3.1 spec → TS types. One source (the Go code), spec is the build artifact published as deliverable 2. |
| Frontend | React + TS + Vite + PWA plugin + React Query | Per spec §2 |
| PDF/CSV report | `maroto`/`gofpdf` (PDF), stdlib `encoding/csv`, `excelize` (XLSX) | IN-2 export |

---

## 2. Repo layout (monorepo)

```
mooring-line-management/
├── api/
│   ├── cmd/server/main.go           # boots HTTP + workers, reads SCOPE/VESSEL_ID
│   ├── internal/
│   │   ├── config/                  # SCOPE=onboard|shore, VESSEL_ID, DB, S3, OIDC
│   │   ├── domain/                  # entities + rules: turning, side accrual, due calcs
│   │   ├── store/                   # pgx repo wrappers, tx helpers (auth.go: users/sessions/flows)
│   │   ├── httpapi/                 # Huma handlers (input/output structs + Register fns); auth.go + authmw.go
│   │   ├── auth/                    # OIDC client (oidc.go), token-at-rest crypto (crypto.go), groups→perms (groups.go)
│   │   ├── storage/                 # S3 client (put/get/presign)
│   │   ├── webhook/                 # subscriptions, HMAC sign, dispatcher
│   │   ├── sync/                    # outbox writer, push/pull worker, conflict rule
│   │   └── report/                  # condition report PDF/CSV/XLSX
│   ├── db/
│   │   ├── migrations/              # NNNN_*.up.sql / .down.sql
│   │   └── queries/                 # *.sql for sqlc
│   ├── openapi/openapi.json         # BUILD ARTIFACT: 3.1 spec emitted from Go (deliverable 2)
│   └── seed/                        # Norwegian Luna seed (~18 lines + spares)
├── web/
│   ├── src/
│   │   ├── api/                     # openapi-typescript types (from api/openapi/openapi.json) + React Query hooks
│   │   ├── app/                     # router, layout, scope-aware nav (vessel switcher on shore)
│   │   ├── features/
│   │   │   ├── dashboard/           # OV-1..3 donut, tiles, attention, trend, feed
│   │   │   ├── deck/                # DK-1..4 deck map, edit layout, drums, rotation
│   │   │   ├── register/            # RP-1..3 table, rope record tabs, add/Ordered
│   │   │   ├── sides/               # TN-1..2 side A/B, turn, due flag
│   │   │   ├── inspections/         # IN-1..3 log form, report, logbook
│   │   │   ├── files/               # IM-1..3 photos timeline, certs/manuals
│   │   │   └── catalogue/           # makers/types/products (shore admin)
│   │   └── lib/                     # search, formatting, offline cache config
│   └── vite.config.ts               # PWA, VITE_SCOPE build flag
├── deploy/
│   ├── docker-compose.onboard.yml   # api(scope=onboard) + postgres + minio
│   ├── docker-compose.shore.yml     # api(scope=shore) + postgres + minio
│   └── Dockerfile
└── README.md
```

---

## 3. Onboard vs shore (one codebase)

- Single binary. `SCOPE=onboard` requires `VESSEL_ID`; `SCOPE=shore` serves all vessels.
- **Scope guard middleware:** onboard injects `vessel_id = $VESSEL_ID` into every query path and rejects cross-vessel access; shore reads `vessel_id` from request/switcher.
- Every row carries `vessel_id` + `origin` marker → same schema serves both (onboard holds 1 vessel, shore holds all).
- Frontend: `VITE_SCOPE` toggles vessel switcher (XC-1) and fleet views.

### Sync (outbox/event replication — preferred per spec)
- `outbox` table: every operational mutation appends a domain event (id, vessel_id, type, payload, created_at, origin).
- **Onboard → shore:** on reconnect, sync worker POSTs unsent operational events to shore `/sync/ingest`; shore applies idempotently (dedupe on event id). Onboard authoritative for ops.
- **Shore → onboard:** shore has its own master-data outbox (catalogue, vessel setup); onboard pulls and applies. Shore authoritative for master.
- Conflict-free by partition: ops are append-only + vessel-owned (onboard wins); master is shore-owned (shore wins).
- Neither side blocks on the other; tolerant of long gaps (cursor/watermark per peer).
- **`app_user` placement:** users authenticate against the external OIDC provider; `app_user` rows are now an upsert-on-login projection of the IdP identity (`oidc_sub`, email, name, groups, `is_admin`), not a local password store — the legacy `password_hash` column was dropped in migration `0008_auth`. Permissions derive from OIDC group claims at request time (`admin` group ⇒ write). Open Q: offline/at-sea authentication onboard — the current BFF flow needs reachability to the IdP at login, so a disconnected-vessel story (cached sessions or an onboard token cache) is still to be designed.

---

## 4. Data model → migrations (FK/constraint highlights)

Tables: `maker, line_type, product, vessel, mooring_station, winch_location, storage_location, drum, mooring_line, line_component(via mooring_line.parent_line_id), certificate, inspection, condition_photo, turn_event, document, webhook_subscription, outbox, app_user`.

**Primary keys (decide before first migration):** every synced table uses **UUID v7** PKs, generated at insert on whichever side creates the row. Required by the sync model — onboard generates rows offline and pushes to shore where all vessels coexist; `bigserial` would collide (vessel A `id=1` vs vessel B `id=1`) at shore aggregation. v7 keeps index locality. `origin` marker + `vessel_id` on every row.

Key constraints (spec §7):
- `serial_number` uniqueness scope = **fleet-wide** (pending NCL confirm). Note the deployment split: a Postgres `UNIQUE` index enforces only **vessel-wide** onboard (it can't see other vessels). Fleet-wide is a **shore-side validation + sync-conflict check**, not a single column constraint meaning the same thing in both deployments. Onboard keeps the vessel-wide UNIQUE; shore adds the global check.
- A line in exactly one location: `current_location_id` + partial logic; **one line per drum** → UNIQUE on `(drum_id)` where occupied (partial unique index).
- Turning only when `can_be_turned` → enforced in domain layer + check on turn endpoint.
- Computed never stored: `total_days_in_service`, side accruals derived in queries.
- `inspection.external_id` UNIQUE per integration → idempotent ingest.
- Side accrual model: store `side_x_accumulated_age_days` (frozen) + `side_x_change_date`; live age = accumulated + (active ? now − change_date : 0). Turn freezes inactive, stamps new active.
- FKs everywhere; `ON DELETE RESTRICT` for catalogue refs, cascade for child photos/components where safe.
- JSONB only for: vessel `layout config` extras, webhook event filters.

---

## 5. Build sequence (vertical slices, API then matching UI)

0. **Foundations** — ✅ codegen de-risk DONE: oapi-codegen + ogen fail on 3.1 nullability; **Huma code-first chosen** (emits 3.1.0, openapi-typescript consumes it). Remaining: repo init, docker-compose (pg+minio), config (SCOPE/VESSEL_ID), golang-migrate tooling, Huma server skeleton + `openapi.json` emit + TS-gen pipeline, auth scaffold, scope guard, health.
1. **Catalogue** (§4.0, shore-owned) — makers, line_types, products, product manual upload. Simplest, master data.
2. **Vessel + layout** (DK-1..4) — stations, winch/storage CRUD, drums 1–6, rotation presets, x/y coords. Needed before lines.
3. **Lines + components + registration + move** (RP-1..3, §4.2/4.3, P0 incl. drum DK-3) — register from product, components→parent, lifecycle incl. Ordered, move winch↔storage with one-line-per-drum guard.
4. **Turning & side tracking** (TN-1..2) — turn endpoint, TurnEvent, accruals, due flag.
5. **Inspections** (IN-1..3) — manual log + `/inspections/ingest` idempotent + logbook. Drives current_condition_status + trend.
6. **Photos & files** (IM-1..3) — S3 upload, condition photo timeline (date/side), certs (per line) vs manuals (per product) separated.
7. **Dashboard + reports + search** (OV-1..3, IN-2, XC-3) — overview aggregate endpoint, condition report PDF/CSV/XLSX worst-first, global search.
8. **Webhooks** — subscriptions + HMAC dispatcher for the 4 event types.
9. **Sync** — outbox + onboard↔shore replication workers + conflict rule.
10. **Seed** — Norwegian Luna ~18 active + spares; runnable both modes.
11. **Cross-cutting UX + PWA** (XC-1/2, §6) — vessel switcher (shore), responsive/tablet, offline read cache + service worker, fallback manual inspection form.

Frontend tracks each slice; React Query hooks generated from spec.

---

## 6. Open questions to confirm with NCL (block exact contracts, not start)
- 3rd-party inspection tool: push or pull? photos inline or URL? line id = our serial or their external id? runs onboard/shore/both? → fixes `/inspections/ingest` contract (build flexible: accept serial OR external_id, photos as refs OR base64).
- Condition scale: assume Good/Monitor/Action (enum, mappable later).
- Serial uniqueness: assume fleet-wide (easy to relax to per-vessel).
- Connectivity/offline-gap profile → sync watermark tuning.
- ~~Auth source: standalone users v1, SSO later~~ → **resolved:** OIDC SSO against the Maranics provider, Backend-for-Frontend. See [authentication.md](authentication.md).
- AMOS integration: later (P2) — keep integration boundary clean.

---

## 7. Deliverables mapping (§11)
1 schema+migrations → §4 · 2 API+OpenAPI+webhooks+upload → §2,8 · 3 PWA all P0 → §5.1-11 · 4 ingest+report export → §5.5,7 · 5 sync → §5.9 · 6 seed → §5.10 · 7 README → final.
```
```
