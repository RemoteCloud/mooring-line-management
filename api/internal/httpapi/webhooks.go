package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/ncl/mooring-api/internal/store"
)

// webhookEventInfo documents a fireable event type and the {{variables}} its payload
// exposes, so the config UI can render checkboxes + a variable cheatsheet.
type webhookEventInfo struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Variables   []string `json:"variables"`
}

// commonVars are available for every event in addition to the per-event payload vars.
var commonVars = []string{"event.type", "event.time", "event.id", "event.aggregate", "event.aggregateId", "vessel.id"}

func withCommon(extra ...string) []string {
	return append(append([]string{}, commonVars...), extra...)
}

var webhookEventCatalog = []webhookEventInfo{
	{"line.registered", "A mooring line was registered", withCommon("payload.id", "payload.serial")},
	{"line.moved", "A line was moved to a drum or storage", withCommon("payload.id", "payload.drumId", "payload.storageId")},
	{"line.turned", "A line was turned end-for-end", withCommon("payload.id", "payload.side")},
	{"inspection.logged", "An inspection was logged (manual or ingested)", withCommon("payload.id", "payload.lineId", "payload.conditionStatus")},
	{"photo.added", "A condition photo was attached to a line", withCommon("payload.id", "payload.lineId", "payload.fileRef")},
	{"document.added", "A document was uploaded for a line", withCommon("payload.id", "payload.lineId", "payload.kind", "payload.fileRef")},
	{"layout.updated", "The deck layout was saved", withCommon("payload.vesselId")},
}

// webhookBody is the create/update request shape. Active is a pointer so an omitted
// value defaults to true on create.
type webhookBody struct {
	Name            string            `json:"name,omitempty"`
	URL             string            `json:"url" minLength:"1" format:"uri" example:"https://example.com/hooks/mooring"`
	Secret          string            `json:"secret,omitempty" doc:"HMAC-SHA256 signing key. Leave empty on update to keep the current secret."`
	Events          []string          `json:"events,omitempty" doc:"Event types to deliver. Empty = all events."`
	Headers         map[string]string `json:"headers,omitempty" doc:"Custom request headers. Values may use {{variable}} substitution."`
	PayloadTemplate string            `json:"payloadTemplate,omitempty" doc:"Body template with {{variable}} substitution. Empty = default JSON envelope."`
	Active          *bool             `json:"active,omitempty"`
}

func (b webhookBody) toStore() store.WebhookSubscription {
	active := true
	if b.Active != nil {
		active = *b.Active
	}
	return store.WebhookSubscription{
		Name:            b.Name,
		URL:             b.URL,
		Secret:          b.Secret,
		Events:          b.Events,
		Headers:         b.Headers,
		PayloadTemplate: b.PayloadTemplate,
		Active:          active,
	}
}

func registerWebhooks(api huma.API, s *Server) {
	tag := []string{"webhooks"}

	huma.Register(api, huma.Operation{
		OperationID: "list-webhook-events", Method: http.MethodGet, Path: "/webhook-events",
		Summary: "List fireable webhook event types", Tags: tag,
	}, func(_ context.Context, _ *struct{}) (*struct{ Body []webhookEventInfo }, error) {
		return &struct{ Body []webhookEventInfo }{Body: webhookEventCatalog}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-webhooks", Method: http.MethodGet, Path: "/webhooks",
		Summary: "List webhook subscriptions", Tags: tag,
	}, func(ctx context.Context, in *struct {
		VesselID string `query:"vesselId"`
	}) (*struct{ Body []store.WebhookSubscription }, error) {
		w, err := s.Store.ListWebhookSubscriptions(ctx, s.vessel(in.VesselID))
		if err != nil {
			return nil, mapErr(err)
		}
		if w == nil {
			w = []store.WebhookSubscription{}
		}
		return &struct{ Body []store.WebhookSubscription }{Body: w}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-webhook", Method: http.MethodPost, Path: "/webhooks",
		Summary: "Create webhook subscription", Tags: tag, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		VesselID string `query:"vesselId"`
		Body     webhookBody
	}) (*struct{ Body store.WebhookSubscription }, error) {
		sub := in.Body.toStore()
		sub.VesselID = s.vessel(in.VesselID)
		w, err := s.Store.CreateWebhookSubscription(ctx, sub)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.WebhookSubscription }{Body: w}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-webhook", Method: http.MethodGet, Path: "/webhooks/{id}",
		Summary: "Get webhook subscription", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body store.WebhookSubscription }, error) {
		w, err := s.Store.GetWebhookSubscription(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.WebhookSubscription }{Body: w}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-webhook", Method: http.MethodPut, Path: "/webhooks/{id}",
		Summary: "Update webhook subscription", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body webhookBody
	}) (*struct{ Body store.WebhookSubscription }, error) {
		w, err := s.Store.UpdateWebhookSubscription(ctx, in.ID, in.Body.toStore())
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.WebhookSubscription }{Body: w}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-webhook", Method: http.MethodDelete, Path: "/webhooks/{id}",
		Summary: "Delete webhook subscription", Tags: tag, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{}, error) {
		if err := s.Store.DeleteWebhookSubscription(ctx, in.ID); err != nil {
			return nil, mapErr(err)
		}
		return &struct{}{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "test-webhook", Method: http.MethodPost, Path: "/webhooks/{id}/test",
		Summary: "Send a test delivery to a webhook", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct {
		Body struct {
			Ok bool `json:"ok"`
		}
	}, error) {
		if err := s.Store.SendTestWebhook(ctx, in.ID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, huma.Error404NotFound("webhook not found")
			}
			return nil, huma.Error502BadGateway("test delivery failed: " + err.Error())
		}
		out := &struct {
			Body struct {
				Ok bool `json:"ok"`
			}
		}{}
		out.Body.Ok = true
		return out, nil
	})
}
