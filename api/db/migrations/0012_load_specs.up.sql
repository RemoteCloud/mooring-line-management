-- Safe Working Load (SWL) and Break Load (MBL), in tonnes, for winches and rope
-- models. A rope inherits these from its product; a winch carries its own.
ALTER TABLE winch_location ADD COLUMN swl double precision, ADD COLUMN break_load double precision;
ALTER TABLE product        ADD COLUMN swl double precision, ADD COLUMN break_load double precision;
