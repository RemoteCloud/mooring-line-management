-- 0016_equipment: received-equipment intake (goods receipt). A lightweight
-- inventory record for gear arriving onboard (lines, shackles, spares): linked to
-- the catalogue product when known, free-text maker/model otherwise, with one-to-many
-- serial numbers, an optional storage location, and photos of the item.

CREATE TABLE equipment (
    id                  uuid PRIMARY KEY,
    vessel_id           uuid NOT NULL REFERENCES vessel (id) ON DELETE CASCADE,
    product_id          uuid REFERENCES product (id) ON DELETE SET NULL,
    name                text NOT NULL,
    maker_text          text,
    model_text          text,
    storage_location_id uuid REFERENCES storage_location (id) ON DELETE SET NULL,
    storage_label       text,
    status              text NOT NULL DEFAULT 'received'
                          CHECK (status IN ('received', 'in_service', 'retired')),
    notes               text,
    received_at         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX equipment_vessel_idx  ON equipment (vessel_id, status);
CREATE INDEX equipment_product_idx ON equipment (product_id);

CREATE TABLE equipment_serial (
    id            uuid PRIMARY KEY,
    equipment_id  uuid NOT NULL REFERENCES equipment (id) ON DELETE CASCADE,
    serial_number text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX equipment_serial_uq ON equipment_serial (equipment_id, serial_number);

CREATE TABLE equipment_photo (
    id           uuid PRIMARY KEY,
    equipment_id uuid NOT NULL REFERENCES equipment (id) ON DELETE CASCADE,
    file_ref     text NOT NULL,
    content_type text,
    caption      text,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX equipment_photo_eq_idx ON equipment_photo (equipment_id);
