package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// WebhookSubscription is an outbound webhook endpoint. The Secret is never
// serialized back to clients (write-only); HasSecret reports whether one is set.
type WebhookSubscription struct {
	ID              string            `json:"id"`
	VesselID        string            `json:"vessel_id,omitempty"`
	Name            string            `json:"name"`
	URL             string            `json:"url"`
	Secret          string            `json:"-"`
	Events          []string          `json:"events"`
	Headers         map[string]string `json:"headers"`
	PayloadTemplate string            `json:"payload_template,omitempty"`
	Active          bool              `json:"active"`
	HasSecret       bool              `json:"has_secret"`
	CreatedAt       time.Time         `json:"created_at"`
}

const webhookCols = `id, COALESCE(vessel_id::text,''), name, url, secret, events, headers, payload_template, active, created_at`

func scanWebhook(row pgx.Row) (WebhookSubscription, error) {
	var w WebhookSubscription
	var headers []byte
	var tmpl *string
	if err := row.Scan(&w.ID, &w.VesselID, &w.Name, &w.URL, &w.Secret, &w.Events, &headers, &tmpl, &w.Active, &w.CreatedAt); err != nil {
		return WebhookSubscription{}, err
	}
	if len(headers) > 0 {
		_ = json.Unmarshal(headers, &w.Headers)
	}
	if w.Headers == nil {
		w.Headers = map[string]string{}
	}
	if w.Events == nil {
		w.Events = []string{}
	}
	if tmpl != nil {
		w.PayloadTemplate = *tmpl
	}
	w.HasSecret = w.Secret != ""
	return w, nil
}

func (s *Store) ListWebhookSubscriptions(ctx context.Context, vesselID string) ([]WebhookSubscription, error) {
	q := `SELECT ` + webhookCols + ` FROM webhook_subscription`
	args := []any{}
	if vesselID != "" {
		q += ` WHERE vessel_id = $1`
		args = append(args, vesselID)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebhookSubscription
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWebhookSubscription(ctx context.Context, id string) (WebhookSubscription, error) {
	return scanWebhook(s.Pool.QueryRow(ctx, `SELECT `+webhookCols+` FROM webhook_subscription WHERE id=$1`, id))
}

func (s *Store) CreateWebhookSubscription(ctx context.Context, w WebhookSubscription) (WebhookSubscription, error) {
	if w.Events == nil {
		w.Events = []string{}
	}
	if w.Headers == nil {
		w.Headers = map[string]string{}
	}
	hb, _ := json.Marshal(w.Headers)
	row := s.Pool.QueryRow(ctx, `
INSERT INTO webhook_subscription (id, vessel_id, name, url, secret, events, headers, payload_template, active)
VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9)
RETURNING `+webhookCols,
		newID(), nullUUID(w.VesselID), w.Name, w.URL, w.Secret, w.Events, hb, w.PayloadTemplate, w.Active)
	return scanWebhook(row)
}

// UpdateWebhookSubscription replaces the mutable fields. An empty Secret keeps the
// existing one (so the UI need not re-enter it on every edit).
func (s *Store) UpdateWebhookSubscription(ctx context.Context, id string, w WebhookSubscription) (WebhookSubscription, error) {
	if w.Events == nil {
		w.Events = []string{}
	}
	if w.Headers == nil {
		w.Headers = map[string]string{}
	}
	hb, _ := json.Marshal(w.Headers)
	row := s.Pool.QueryRow(ctx, `
UPDATE webhook_subscription SET
  name=$2, url=$3,
  secret=COALESCE(NULLIF($4,''), secret),
  events=$5, headers=$6, payload_template=NULLIF($7,''), active=$8, updated_at=now()
WHERE id=$1
RETURNING `+webhookCols,
		id, w.Name, w.URL, w.Secret, w.Events, hb, w.PayloadTemplate, w.Active)
	return scanWebhook(row)
}

func (s *Store) DeleteWebhookSubscription(ctx context.Context, id string) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM webhook_subscription WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
