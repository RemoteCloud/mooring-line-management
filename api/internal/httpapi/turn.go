package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/store"
)

// turnBody carries the optional note recorded with a turn.
type turnBody struct {
	Note string `json:"note,omitempty"`
}

// registerTurn wires the turn-line operation. The orchestrator calls this from
// api.go; it is intentionally not invoked here.
func registerTurn(api huma.API, s *Server) {
	huma.Register(api, huma.Operation{
		OperationID: "turn-line", Method: http.MethodPost, Path: "/lines/{id}/turn",
		Summary: "Turn a line to its other side", Tags: []string{"turning"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body turnBody
	}) (*struct{ Body store.Line }, error) {
		l, err := s.Store.TurnLine(ctx, in.ID, in.Body.Note)
		if err != nil {
			if errors.Is(err, store.ErrNotTurnable) {
				return nil, huma.Error422UnprocessableEntity("line is not turnable")
			}
			return nil, mapErr(err)
		}
		return &struct{ Body store.Line }{Body: l}, nil
	})
}
