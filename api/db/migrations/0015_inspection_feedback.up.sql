-- Feedback channel on an existing inspection: lets a 3rd-party system (or crew)
-- attach a follow-up assessment/acknowledgement to an inspection. Annotation
-- only; does not change the line's condition.
CREATE TABLE inspection_feedback (
    id               uuid PRIMARY KEY,
    inspection_id    uuid NOT NULL REFERENCES inspection (id) ON DELETE CASCADE,
    external_id      text,                       -- 3rd-party idempotency key
    source           text NOT NULL DEFAULT 'api' CHECK (source IN ('api', 'manual')),
    author           text,                       -- who/what gave the feedback
    status           text NOT NULL CHECK (status IN ('acknowledged', 'disputed', 'resolved', 'comment')),
    condition_status text CHECK (condition_status IN ('Good', 'Monitor', 'Action')), -- optional suggested condition
    notes            text,
    created_at       timestamptz NOT NULL DEFAULT now()
);

-- Idempotency: a duplicate external_id is ignored on insert.
CREATE UNIQUE INDEX inspection_feedback_external_key ON inspection_feedback (external_id) WHERE external_id IS NOT NULL;
CREATE INDEX inspection_feedback_insp_idx ON inspection_feedback (inspection_id, created_at);
