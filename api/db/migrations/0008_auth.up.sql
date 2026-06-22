-- 0008_auth: OIDC authentication. Removes the legacy password-based auth scaffold
-- (never wired up) and introduces external OpenID Connect identity + server-side
-- session storage (Backend-for-Frontend: tokens live here, never in the browser).

-- Repurpose app_user for OIDC identities. Password auth is gone.
ALTER TABLE app_user DROP COLUMN IF EXISTS password_hash;
-- The role CHECK constraint encoded the old fixed role set; permissions now derive
-- from OIDC groups. Keep the column as free-text (nullable) for display/legacy.
ALTER TABLE app_user DROP CONSTRAINT IF EXISTS app_user_role_check;
ALTER TABLE app_user ALTER COLUMN role DROP NOT NULL;

ALTER TABLE app_user ADD COLUMN oidc_sub      text UNIQUE;
ALTER TABLE app_user ADD COLUMN groups        text;                         -- JSON array of group names
ALTER TABLE app_user ADD COLUMN is_admin      boolean NOT NULL DEFAULT false;
ALTER TABLE app_user ADD COLUMN last_login_at timestamptz;

-- Short-lived OIDC auth-code flow state (PKCE verifier, nonce, return target).
-- One row per in-flight login; deleted on callback; expired rows are ignored.
CREATE TABLE oidc_flow (
    state         text PRIMARY KEY,
    code_verifier text NOT NULL,
    nonce         text NOT NULL,
    return_to     text NOT NULL DEFAULT '/',
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Server-side sessions. The browser holds only the opaque sid (HttpOnly cookie);
-- the encrypted OAuth/OIDC tokens stay here.
CREATE TABLE auth_session (
    sid               text PRIMARY KEY,
    user_id           uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    access_token_enc  text,
    refresh_token_enc text,
    id_token_enc      text,
    access_expires_at timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    last_seen_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX auth_session_user_idx ON auth_session (user_id);
