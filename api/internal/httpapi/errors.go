package httpapi

import (
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/ncl/mooring-api/internal/store"
)

// mapErr converts store/db errors into appropriate HTTP errors.
func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return huma.Error404NotFound("not found")
	case errors.Is(err, store.ErrDrumOccupied):
		return huma.Error409Conflict("drum already holds a line")
	case errors.Is(err, store.ErrOccupied):
		return huma.Error409Conflict("position still holds a line")
	case errors.Is(err, store.ErrInvalidMoveTarget):
		return huma.Error422UnprocessableEntity("invalid move target: name exactly one destination on this vessel")
	default:
		return huma.Error500InternalServerError("internal error", err)
	}
}
