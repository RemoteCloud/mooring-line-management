-- Revert to the original three document kinds. Fails if any 'delivery' rows remain.
ALTER TABLE document DROP CONSTRAINT IF EXISTS document_kind_check;
ALTER TABLE document ADD CONSTRAINT document_kind_check
    CHECK (kind IN ('certificate', 'manual', 'guide'));
