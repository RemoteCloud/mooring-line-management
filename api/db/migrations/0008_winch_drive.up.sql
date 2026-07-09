-- Winch drive type (electric/hydraulic) and whether the label is auto-derived
-- from the winch's deck position (station + port/starboard) or hand-edited.
ALTER TABLE winch_location
    ADD COLUMN drive_type text NOT NULL DEFAULT 'electric'
        CHECK (drive_type IN ('electric', 'hydraulic'));

ALTER TABLE winch_location
    ADD COLUMN label_auto boolean NOT NULL DEFAULT false;
