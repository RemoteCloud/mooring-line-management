-- Reverse 0008_auth. Restores the legacy password/role scaffold so the schema
-- matches the pre-OIDC state. (password_hash is NOT NULL in 0001; we give a
-- default of '' on restore since rows may exist.)

DROP TABLE IF EXISTS auth_session;
DROP TABLE IF EXISTS oidc_flow;

ALTER TABLE app_user DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE app_user DROP COLUMN IF EXISTS is_admin;
ALTER TABLE app_user DROP COLUMN IF EXISTS groups;
ALTER TABLE app_user DROP COLUMN IF EXISTS oidc_sub;

ALTER TABLE app_user ADD COLUMN IF NOT EXISTS password_hash text NOT NULL DEFAULT '';
ALTER TABLE app_user ALTER COLUMN password_hash DROP DEFAULT;
ALTER TABLE app_user ALTER COLUMN role SET NOT NULL;
ALTER TABLE app_user ADD CONSTRAINT app_user_role_check
    CHECK (role IN ('admin', 'vessel_user', 'readonly'));
