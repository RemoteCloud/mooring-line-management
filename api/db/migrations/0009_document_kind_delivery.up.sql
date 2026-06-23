-- 0009_document_kind_delivery: allow 'delivery' documents (delivery notes captured
-- at line registration) alongside certificates/manuals/guides.
ALTER TABLE document DROP CONSTRAINT IF EXISTS document_kind_check;
ALTER TABLE document ADD CONSTRAINT document_kind_check
    CHECK (kind IN ('certificate', 'manual', 'guide', 'delivery'));
