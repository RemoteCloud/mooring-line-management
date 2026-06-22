-- The user's UserManagement POSITION, read from /userinfo at login. position_id
-- drives admin (matched against OIDC_ADMIN_GROUP); position_name is for display.
ALTER TABLE app_user ADD COLUMN IF NOT EXISTS position_id   text;
ALTER TABLE app_user ADD COLUMN IF NOT EXISTS position_name text;
