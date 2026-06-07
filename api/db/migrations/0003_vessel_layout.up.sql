-- 0003_vessel_layout: vessels and their deck layout (stations, winches, drums, storage).
-- Station is fwd/aft, modelled as a column rather than a separate table.

CREATE TABLE vessel (
    id         uuid PRIMARY KEY,
    name       text NOT NULL,
    imo        text,
    fleet_id   uuid,
    origin     text NOT NULL DEFAULT 'shore',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE winch_location (
    id          uuid PRIMARY KEY,
    vessel_id   uuid NOT NULL REFERENCES vessel (id) ON DELETE CASCADE,
    label       text NOT NULL,
    station     text NOT NULL CHECK (station IN ('fwd', 'aft')),
    x           real NOT NULL DEFAULT 0.5,          -- normalized 0..1 for responsive rendering
    y           real NOT NULL DEFAULT 0.5,
    orientation int  NOT NULL DEFAULT 0 CHECK (orientation IN (0, 45, -45, 90, -90)),
    drum_count  int  NOT NULL DEFAULT 1 CHECK (drum_count BETWEEN 1 AND 6),
    origin      text NOT NULL DEFAULT 'onboard',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX winch_vessel_idx ON winch_location (vessel_id, station);

CREATE TABLE storage_location (
    id         uuid PRIMARY KEY,
    vessel_id  uuid NOT NULL REFERENCES vessel (id) ON DELETE CASCADE,
    label      text NOT NULL,
    station    text NOT NULL CHECK (station IN ('fwd', 'aft')),
    x          real NOT NULL DEFAULT 0.5,
    y          real NOT NULL DEFAULT 0.5,
    origin     text NOT NULL DEFAULT 'onboard',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX storage_vessel_idx ON storage_location (vessel_id, station);

CREATE TABLE drum (
    id       uuid PRIMARY KEY,
    winch_id uuid NOT NULL REFERENCES winch_location (id) ON DELETE CASCADE,
    idx      int  NOT NULL CHECK (idx BETWEEN 1 AND 6),
    UNIQUE (winch_id, idx)
);
CREATE INDEX drum_winch_idx ON drum (winch_id);
