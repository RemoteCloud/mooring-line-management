-- 0002_catalogue: master/reference data (shore-owned). A Maker makes Products;
-- each Product is of one LineType. Registered lines are instances of a Product.

CREATE TABLE maker (
    id         uuid PRIMARY KEY,
    name       text NOT NULL,
    notes      text,
    origin     text NOT NULL DEFAULT 'shore',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX maker_name_key ON maker (lower(name));

CREATE TABLE line_type (
    id          uuid PRIMARY KEY,
    name        text NOT NULL,
    description text,
    origin      text NOT NULL DEFAULT 'shore',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX line_type_name_key ON line_type (lower(name));

CREATE TABLE product (
    id                      uuid PRIMARY KEY,
    maker_id                uuid NOT NULL REFERENCES maker (id) ON DELETE RESTRICT,
    line_type_id            uuid NOT NULL REFERENCES line_type (id) ON DELETE RESTRICT,
    product_name            text NOT NULL,
    construction_type       text,
    default_length          double precision,   -- nullable; actual length is per instance
    can_be_turned           boolean NOT NULL DEFAULT true,
    manufacturer_manual_ref text,               -- manual belongs to the product
    notes                   text,
    origin                  text NOT NULL DEFAULT 'shore',
    created_at              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX product_maker_idx ON product (maker_id);
CREATE INDEX product_line_type_idx ON product (line_type_id);
