-- Off-map (text-based) storage areas: named locations like "Under mooring deck"
-- or "Deck side shelf" that hold ropes but aren't drawn on the deck plan. They have
-- no x/y meaning and no station (vessel-wide), so station becomes optional.
ALTER TABLE storage_location ADD COLUMN on_map boolean NOT NULL DEFAULT true;
ALTER TABLE storage_location ALTER COLUMN station DROP NOT NULL;
ALTER TABLE storage_location DROP CONSTRAINT storage_location_station_check;
ALTER TABLE storage_location ADD CONSTRAINT storage_location_station_check
    CHECK (station IS NULL OR station IN ('fwd', 'aft'));
