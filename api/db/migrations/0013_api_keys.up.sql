-- 0013_api_keys: basic API-key auth (intentionally temporary; to be replaced).
-- Keys are stored as sha-256 hashes only — never the plaintext.

-- Under key-only auth nothing writes app_user.password_hash; default it so inserts can omit it.
ALTER TABLE app_user ALTER COLUMN password_hash SET DEFAULT '';

CREATE TABLE api_key (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    name         text NOT NULL,                 -- human label, e.g. "Bridge tablet"
    key_hash     text NOT NULL,                 -- hex sha-256 of the full presented key
    key_prefix   text NOT NULL,                 -- "mlm_" + first chars, display only
    last_used_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    revoked_at   timestamptz                    -- null = active
);

CREATE UNIQUE INDEX api_key_hash_key ON api_key (key_hash);
CREATE INDEX api_key_user_idx ON api_key (user_id);
-- Hot path: authenticate by hash, active keys only.
CREATE INDEX api_key_active_idx ON api_key (key_hash) WHERE revoked_at IS NULL;
