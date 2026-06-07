-- 0005_turning: turn / position-change history for lines.

CREATE TABLE turn_event (
    id               uuid PRIMARY KEY,
    line_id          uuid NOT NULL REFERENCES mooring_line (id) ON DELETE CASCADE,
    event_type       text NOT NULL CHECK (event_type IN ('turn', 'position_change')),
    event_date       date NOT NULL,
    from_location_id uuid,
    to_location_id   uuid,
    side_after       text CHECK (side_after IN ('A', 'B', 'n/a')),
    note             text,
    origin           text NOT NULL DEFAULT 'onboard',
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX turn_event_line_idx ON turn_event (line_id, event_date);
