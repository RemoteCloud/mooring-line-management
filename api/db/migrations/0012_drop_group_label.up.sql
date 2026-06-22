-- Group names are now resolved live from UserManagement (positionTeams API), so
-- the manual per-group label is redundant.
ALTER TABLE group_access DROP COLUMN IF EXISTS label;
