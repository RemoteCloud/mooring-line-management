-- 0004_lines: mooring lines (instances of a Product) and their components.

CREATE TABLE mooring_line (
    id               uuid PRIMARY KEY,
    vessel_id        uuid NOT NULL REFERENCES vessel (id) ON DELETE CASCADE,
    product_id       uuid NOT NULL REFERENCES product (id) ON DELETE RESTRICT,

    name             text NOT NULL,
    tag_number       text,
    certificate_number text,
    serial_number    text NOT NULL,
    lifecycle_status text NOT NULL DEFAULT 'active'
                          CHECK (lifecycle_status IN ('ordered', 'active', 'spare', 'retired')),

    length             double precision,
    manufacture_date   date,
    installation_date  date,
    can_be_turned      boolean NOT NULL DEFAULT true,

    -- side tracking
    current_side             text CHECK (current_side IN ('A', 'B', 'n/a')),
    side_a_change_date       date,
    side_a_accumulated_age_days int NOT NULL DEFAULT 0,
    side_a_condition         text CHECK (side_a_condition IN ('Good', 'Monitor', 'Action')),
    side_b_change_date       date,
    side_b_accumulated_age_days int NOT NULL DEFAULT 0,
    side_b_condition         text CHECK (side_b_condition IN ('Good', 'Monitor', 'Action')),

    current_condition_status text CHECK (current_condition_status IN ('Good', 'Monitor', 'Action')),
    certificate_ref          text,

    -- location: at most one of drum / storage (see CHECK). Spare = in storage; ordered = neither yet.
    current_drum_id    uuid REFERENCES drum (id) ON DELETE SET NULL,
    current_storage_id uuid REFERENCES storage_location (id) ON DELETE SET NULL,

    parent_line_id uuid REFERENCES mooring_line (id) ON DELETE CASCADE,  -- set for components

    origin     text NOT NULL DEFAULT 'onboard',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT one_location_at_most
        CHECK (NOT (current_drum_id IS NOT NULL AND current_storage_id IS NOT NULL))
);

-- serial unique: enforces vessel-wide onboard (one vessel per DB) and fleet-wide on shore.
CREATE UNIQUE INDEX mooring_line_serial_key ON mooring_line (serial_number);

-- one line per drum
CREATE UNIQUE INDEX mooring_line_drum_key ON mooring_line (current_drum_id)
    WHERE current_drum_id IS NOT NULL;

CREATE INDEX mooring_line_vessel_idx ON mooring_line (vessel_id);
CREATE INDEX mooring_line_parent_idx ON mooring_line (parent_line_id);
CREATE INDEX mooring_line_product_idx ON mooring_line (product_id);
