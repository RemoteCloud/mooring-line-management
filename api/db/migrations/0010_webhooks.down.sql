DROP TABLE IF EXISTS webhook_delivery;

ALTER TABLE webhook_subscription
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS headers,
    DROP COLUMN IF EXISTS payload_template,
    DROP COLUMN IF EXISTS updated_at;
