-- Reverse off-map storage: drop any position-less areas back onto a station so the
-- restored NOT NULL holds, then remove the flag.
UPDATE storage_location SET station = 'fwd' WHERE station IS NULL;
ALTER TABLE storage_location DROP CONSTRAINT storage_location_station_check;
ALTER TABLE storage_location ALTER COLUMN station SET NOT NULL;
ALTER TABLE storage_location ADD CONSTRAINT storage_location_station_check
    CHECK (station IN ('fwd', 'aft'));
ALTER TABLE storage_location DROP COLUMN on_map;
