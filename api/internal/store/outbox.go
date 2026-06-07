package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// writeOutbox appends a domain event in the same transaction as the mutation that
// produced it. Drives webhook dispatch and onboard<->shore sync (later slices).
func writeOutbox(ctx context.Context, tx pgx.Tx, vesselID, aggregate, aggregateID, eventType string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO outbox (id, vessel_id, aggregate, aggregate_id, event_type, payload, origin)
VALUES ($1,$2,$3,$4,$5,$6,'onboard')`,
		uuid.Must(uuid.NewV7()).String(), nullUUID(vesselID), aggregate, nullUUID(aggregateID), eventType, b)
	return err
}
