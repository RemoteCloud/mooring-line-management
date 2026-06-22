-- 0010_group_access: group-based access control. Maps OIDC group ids (opaque
-- GUIDs from the provider) to an access level. A row's presence grants access at
-- its level; the ABSENCE of a row means "denied" (no access). Admins are resolved
-- separately (group/email allowlist), not via this table.

CREATE TABLE group_access (
    group_id   text PRIMARY KEY,
    level      text NOT NULL CHECK (level IN ('view','edit')),
    label      text,
    updated_by text,
    updated_at timestamptz NOT NULL DEFAULT now()
);
