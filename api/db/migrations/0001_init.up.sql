-- 0001_init: cross-cutting infrastructure shared by every slice.
-- Domain tables (catalogue, vessel, lines, ...) arrive in their own migrations.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Application users. Shore-owned master data (authored on shore, synced down) so
-- crew can authenticate onboard while disconnected. See PLAN.md §3.
CREATE TABLE app_user (
    id            uuid PRIMARY KEY,
    email         text NOT NULL,
    name          text NOT NULL,
    role          text NOT NULL CHECK (role IN ('admin', 'vessel_user', 'readonly')),
    password_hash text NOT NULL,
    vessel_id     uuid,                       -- null = fleet/shore user; set = scoped to a vessel
    active        boolean NOT NULL DEFAULT true,
    origin        text NOT NULL DEFAULT 'shore',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX app_user_email_key ON app_user (lower(email));

-- Outbox: every operational mutation appends an event here. Drives both webhook
-- dispatch and onboard<->shore sync replication. Append-only.
CREATE TABLE outbox (
    id           uuid PRIMARY KEY,
    vessel_id    uuid,                        -- the vessel an event belongs to (null for fleet master data)
    aggregate    text NOT NULL,               -- e.g. 'mooring_line', 'inspection'
    aggregate_id uuid,
    event_type   text NOT NULL,               -- e.g. 'inspection.logged', 'line.turned'
    payload      jsonb NOT NULL,
    origin       text NOT NULL,               -- 'onboard' | 'shore'
    created_at   timestamptz NOT NULL DEFAULT now(),
    webhook_published_at timestamptz,         -- null until dispatched to webhooks
    synced_at    timestamptz                  -- null until replicated to the peer deployment
);
CREATE INDEX outbox_unpublished_idx ON outbox (created_at) WHERE webhook_published_at IS NULL;
CREATE INDEX outbox_unsynced_idx ON outbox (created_at) WHERE synced_at IS NULL;
CREATE INDEX outbox_vessel_idx ON outbox (vessel_id, created_at);

-- Webhook subscriptions (HMAC-signed delivery). Subscribable per vessel or fleet.
CREATE TABLE webhook_subscription (
    id         uuid PRIMARY KEY,
    vessel_id  uuid,                          -- null = all vessels (fleet subscription, shore only)
    url        text NOT NULL,
    secret     text NOT NULL,                 -- HMAC-SHA256 signing key
    events     text[] NOT NULL DEFAULT '{}',  -- empty = all event types
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);
