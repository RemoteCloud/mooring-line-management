-- Extend the webhook_subscription skeleton with a display name, custom headers, a
-- payload template, and an updated timestamp so the dispatcher can render per-webhook
-- bodies and headers with {{variable}} substitution.
ALTER TABLE webhook_subscription
    ADD COLUMN name             text NOT NULL DEFAULT '',
    ADD COLUMN headers          jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN payload_template text,
    ADD COLUMN updated_at       timestamptz NOT NULL DEFAULT now();

-- Per-(event, subscription) delivery tracking: lets the poller fan a single outbox
-- event out to many subscriptions, retry each independently with backoff, and record
-- the last failure. The single outbox.webhook_published_at flag can only mark that an
-- event has been fanned out, not the per-subscription outcome.
CREATE TABLE webhook_delivery (
    id              uuid PRIMARY KEY,
    event_id        uuid NOT NULL REFERENCES outbox(id) ON DELETE CASCADE,
    subscription_id uuid NOT NULL REFERENCES webhook_subscription(id) ON DELETE CASCADE,
    status          text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','delivered','failed')),
    attempts        int  NOT NULL DEFAULT 0,
    last_error      text,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    delivered_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (event_id, subscription_id)
);

-- Due-work index for the dispatcher poll: pending/failed rows ready to (re)send.
CREATE INDEX webhook_delivery_due_idx ON webhook_delivery (next_attempt_at)
    WHERE status IN ('pending', 'failed');
