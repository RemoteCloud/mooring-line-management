-- 0007_files: documents (certificates, manuals, guides) attached to a line or product.

CREATE TABLE document (
    id           uuid PRIMARY KEY,
    line_id      uuid REFERENCES mooring_line (id) ON DELETE CASCADE,
    product_id   uuid REFERENCES product (id) ON DELETE CASCADE,
    vessel_id    uuid REFERENCES vessel (id) ON DELETE CASCADE,
    kind         text NOT NULL CHECK (kind IN ('certificate', 'manual', 'guide')),
    file_ref     text NOT NULL,
    file_name    text NOT NULL,
    content_type text,
    size_bytes   bigint,
    origin       text NOT NULL DEFAULT 'onboard',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX document_line_idx ON document (line_id);
CREATE INDEX document_product_idx ON document (product_id);
CREATE INDEX document_vessel_idx ON document (vessel_id);
