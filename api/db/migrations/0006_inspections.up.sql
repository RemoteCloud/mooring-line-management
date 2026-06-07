-- 0006_inspections: inspection events + condition photos.

CREATE TABLE inspection (
    id               uuid PRIMARY KEY,
    line_id          uuid NOT NULL REFERENCES mooring_line (id) ON DELETE CASCADE,
    vessel_id        uuid NOT NULL REFERENCES vessel (id) ON DELETE CASCADE,
    inspected_at     timestamptz NOT NULL,
    inspected_by     text,
    source           text NOT NULL DEFAULT 'manual' CHECK (source IN ('api', 'manual')),
    external_id      text,                    -- supplied by API ingest for idempotency
    condition_status text NOT NULL CHECK (condition_status IN ('Good', 'Monitor', 'Action')),
    notes            text,
    origin           text NOT NULL DEFAULT 'onboard',
    created_at       timestamptz NOT NULL DEFAULT now()
);

-- idempotent ingest: an external_id may appear at most once (when present).
CREATE UNIQUE INDEX inspection_external_key ON inspection (external_id)
    WHERE external_id IS NOT NULL;

CREATE INDEX inspection_line_idx ON inspection (line_id, inspected_at);
CREATE INDEX inspection_vessel_idx ON inspection (vessel_id, inspected_at);

CREATE TABLE condition_photo (
    id                  uuid PRIMARY KEY,
    line_id             uuid NOT NULL REFERENCES mooring_line (id) ON DELETE CASCADE,
    inspection_id       uuid REFERENCES inspection (id) ON DELETE SET NULL,
    file_ref            text NOT NULL,
    taken_at            date,
    side                text CHECK (side IN ('A', 'B', 'n/a')),
    condition_at_capture text CHECK (condition_at_capture IN ('Good', 'Monitor', 'Action')),
    origin              text NOT NULL DEFAULT 'onboard',
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX condition_photo_line_idx ON condition_photo (line_id, taken_at);
CREATE INDEX condition_photo_inspection_idx ON condition_photo (inspection_id);
