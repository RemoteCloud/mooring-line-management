package store

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// webhookClient is shared by the dispatcher and the synchronous test send.
var webhookClient = &http.Client{Timeout: 15 * time.Second}

const (
	webhookBatch       = 50
	webhookMaxAttempts = 10
)

// outboxEvent is a domain event as stored in the outbox, used to render a delivery.
type outboxEvent struct {
	ID          string
	VesselID    string
	Aggregate   string
	AggregateID string
	EventType   string
	Payload     []byte
	CreatedAt   time.Time
}

// dueDelivery is a webhook_delivery row joined with its subscription + event,
// ready to send.
type dueDelivery struct {
	DeliveryID string
	Attempts   int
	URL        string
	Secret     string
	Headers    map[string]string
	Template   string
	Event      outboxEvent
}

// RunWebhookDispatcher polls the outbox, fans new events out to matching
// subscriptions, and (re)sends due deliveries until ctx is cancelled.
func (s *Store) RunWebhookDispatcher(ctx context.Context, log *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	log.Info("webhook dispatcher started")
	for {
		select {
		case <-ctx.Done():
			log.Info("webhook dispatcher stopped")
			return
		case <-ticker.C:
			if err := s.dispatchTick(ctx, log); err != nil {
				log.Warn("webhook dispatch tick failed", "err", err)
			}
		}
	}
}

func (s *Store) dispatchTick(ctx context.Context, log *slog.Logger) error {
	if err := s.enqueuePendingDeliveries(ctx); err != nil {
		return err
	}
	due, err := s.claimDueDeliveries(ctx, webhookBatch)
	if err != nil {
		return err
	}
	for _, d := range due {
		body, headers := renderDelivery(d)
		if err := postWebhook(ctx, d, body, headers); err != nil {
			if merr := s.markFailed(ctx, d.DeliveryID, d.Attempts, err.Error()); merr != nil {
				log.Error("webhook markFailed failed", "id", d.DeliveryID, "err", merr)
			}
			log.Warn("webhook delivery failed", "url", d.URL, "event", d.Event.EventType, "attempt", d.Attempts+1, "err", err)
		} else if merr := s.markDelivered(ctx, d.DeliveryID); merr != nil {
			log.Error("webhook markDelivered failed", "id", d.DeliveryID, "err", merr)
		}
	}
	return nil
}

// enqueuePendingDeliveries creates delivery rows for every unpublished outbox event
// against each matching active subscription, then marks those events published so we
// never re-fan them (a subscription added later only receives future events).
func (s *Store) enqueuePendingDeliveries(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
INSERT INTO webhook_delivery (id, event_id, subscription_id)
SELECT gen_random_uuid(), o.id, ws.id
FROM outbox o
JOIN webhook_subscription ws
  ON ws.active
 AND (ws.vessel_id IS NULL OR ws.vessel_id = o.vessel_id)
 AND (cardinality(ws.events) = 0 OR o.event_type = ANY(ws.events))
WHERE o.webhook_published_at IS NULL
ON CONFLICT (event_id, subscription_id) DO NOTHING`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE outbox SET webhook_published_at = now() WHERE webhook_published_at IS NULL`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) claimDueDeliveries(ctx context.Context, limit int) ([]dueDelivery, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT d.id, d.attempts, ws.url, ws.secret, ws.headers, COALESCE(ws.payload_template,''),
       o.id, COALESCE(o.vessel_id::text,''), o.aggregate, COALESCE(o.aggregate_id::text,''),
       o.event_type, o.payload, o.created_at
FROM webhook_delivery d
JOIN webhook_subscription ws ON ws.id = d.subscription_id
JOIN outbox o ON o.id = d.event_id
WHERE d.status IN ('pending','failed') AND d.next_attempt_at <= now() AND ws.active
ORDER BY d.next_attempt_at
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dueDelivery
	for rows.Next() {
		var d dueDelivery
		var headers []byte
		if err := rows.Scan(&d.DeliveryID, &d.Attempts, &d.URL, &d.Secret, &headers, &d.Template,
			&d.Event.ID, &d.Event.VesselID, &d.Event.Aggregate, &d.Event.AggregateID,
			&d.Event.EventType, &d.Event.Payload, &d.Event.CreatedAt); err != nil {
			return nil, err
		}
		if len(headers) > 0 {
			_ = json.Unmarshal(headers, &d.Headers)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) markDelivered(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE webhook_delivery SET status='delivered', delivered_at=now(), attempts=attempts+1, last_error=NULL WHERE id=$1`, id)
	return err
}

func (s *Store) markFailed(ctx context.Context, id string, attempts int, msg string) error {
	delay := backoffSeconds(attempts)
	if attempts+1 >= webhookMaxAttempts {
		delay = 100 * 365 * 24 * 3600 // give up: park far in the future, keep the row for audit
	}
	_, err := s.Pool.Exec(ctx, `
UPDATE webhook_delivery
SET status='failed', attempts=attempts+1, last_error=$2,
    next_attempt_at = now() + make_interval(secs => $3)
WHERE id=$1`, id, msg, delay)
	return err
}

// backoffSeconds grows 30s, 60s, 120s … capped at 1h.
func backoffSeconds(attempts int) int {
	d := 30 << attempts
	if d > 3600 || d <= 0 {
		d = 3600
	}
	return d
}

// ---- rendering ----

var tmplVar = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

// renderTemplate substitutes {{dotted.key}} tokens from vars; unknown keys become "".
func renderTemplate(tmpl string, vars map[string]string) string {
	return tmplVar.ReplaceAllStringFunc(tmpl, func(m string) string {
		key := tmplVar.FindStringSubmatch(m)[1]
		return vars[key]
	})
}

// deliveryVars flattens an event into the {{variable}} namespace: event.*, vessel.id,
// and each top-level payload field as payload.<key>.
func deliveryVars(e outboxEvent) map[string]string {
	v := map[string]string{
		"event.id":           e.ID,
		"event.type":         e.EventType,
		"event.time":         e.CreatedAt.UTC().Format(time.RFC3339),
		"event.aggregate":   e.Aggregate,
		"event.aggregateId": e.AggregateID,
		"vessel.id":         e.VesselID,
	}
	var pl map[string]any
	if json.Unmarshal(e.Payload, &pl) == nil {
		for k, val := range pl {
			v["payload."+k] = fmt.Sprint(val)
		}
	}
	return v
}

func defaultBody(e outboxEvent) []byte {
	b, _ := json.Marshal(map[string]any{
		"event":       e.EventType,
		"eventId":     e.ID,
		"vesselId":    e.VesselID,
		"aggregate":   e.Aggregate,
		"aggregateId": e.AggregateID,
		"createdAt":   e.CreatedAt.UTC().Format(time.RFC3339),
		"data":        json.RawMessage(e.Payload),
	})
	return b
}

func renderDelivery(d dueDelivery) ([]byte, map[string]string) {
	vars := deliveryVars(d.Event)
	var body []byte
	if strings.TrimSpace(d.Template) != "" {
		body = []byte(renderTemplate(d.Template, vars))
	} else {
		body = defaultBody(d.Event)
	}
	headers := make(map[string]string, len(d.Headers))
	for k, val := range d.Headers {
		headers[k] = renderTemplate(val, vars)
	}
	return body, headers
}

func postWebhook(ctx context.Context, d dueDelivery, body []byte, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "mooring-webhook/1")
	req.Header.Set("X-Webhook-Event", d.Event.EventType)
	for k, v := range headers { // custom headers win (may override Content-Type)
		req.Header.Set(k, v)
	}
	if d.Secret != "" {
		mac := hmac.New(sha256.New, []byte(d.Secret))
		mac.Write(body)
		req.Header.Set("X-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := webhookClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}
	return nil
}

// SendTestWebhook delivers a synthetic webhook.test event to one subscription so the
// config UI can validate URL + headers + template without waiting for a real event.
func (s *Store) SendTestWebhook(ctx context.Context, id string) error {
	w, err := s.GetWebhookSubscription(ctx, id)
	if err != nil {
		return err
	}
	d := dueDelivery{
		URL:      w.URL,
		Secret:   w.Secret,
		Headers:  w.Headers,
		Template: w.PayloadTemplate,
		Event: outboxEvent{
			ID:        newID(),
			VesselID:  w.VesselID,
			Aggregate: "test",
			EventType: "webhook.test",
			Payload:   []byte(`{"message":"This is a test delivery from Mooring Line Management.","ok":true}`),
			CreatedAt: time.Now().UTC(),
		},
	}
	body, headers := renderDelivery(d)
	return postWebhook(ctx, d, body, headers)
}
