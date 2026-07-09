package httpapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/store"
)

// registerUsers wires basic (temporary) user + API-key management. Every endpoint except
// GET /me requires an admin key; /me returns the caller's own identity.
func registerUsers(api huma.API, s *Server) {
	tag := []string{"users"}

	huma.Register(api, huma.Operation{
		OperationID: "whoami", Method: http.MethodGet, Path: "/me",
		Summary: "Current authenticated user", Tags: tag,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body store.AuthUser }, error) {
		u, ok := AuthedUser(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		return &struct{ Body store.AuthUser }{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-users", Method: http.MethodGet, Path: "/users",
		Summary: "List users", Tags: tag,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.User }, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		u, err := s.Store.ListUsers(ctx)
		if err != nil {
			return nil, mapErr(err)
		}
		if u == nil {
			u = []store.User{}
		}
		return &struct{ Body []store.User }{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-user", Method: http.MethodPost, Path: "/users",
		Summary: "Create user", Tags: tag, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		Body struct {
			Email    string `json:"email" minLength:"1" format:"email"`
			Name     string `json:"name" minLength:"1"`
			Role     string `json:"role" enum:"admin,vessel_user,readonly"`
			VesselID string `json:"vesselId,omitempty" format:"uuid"`
		}
	}) (*struct{ Body store.User }, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		u, err := s.Store.CreateUser(ctx, in.Body.Email, in.Body.Name, in.Body.Role, in.Body.VesselID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.User }{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-user", Method: http.MethodPatch, Path: "/users/{id}",
		Summary: "Update user (active/role)", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			Active *bool   `json:"active,omitempty"`
			Role   *string `json:"role,omitempty" enum:"admin,vessel_user,readonly"`
		}
	}) (*struct{ Body store.User }, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		u, err := s.Store.UpdateUser(ctx, in.ID, in.Body.Active, in.Body.Role)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.User }{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-api-keys", Method: http.MethodGet, Path: "/users/{id}/api-keys",
		Summary: "List a user's API keys (metadata only)", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body []store.APIKey }, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		k, err := s.Store.ListAPIKeys(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		if k == nil {
			k = []store.APIKey{}
		}
		return &struct{ Body []store.APIKey }{Body: k}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-api-key", Method: http.MethodPost, Path: "/users/{id}/api-keys",
		Summary: "Issue an API key (plaintext returned once)", Tags: tag, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			Name string `json:"name" minLength:"1"`
		}
	}) (*struct{ Body store.NewAPIKey }, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		k, err := s.Store.CreateAPIKey(ctx, in.ID, in.Body.Name)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.NewAPIKey }{Body: k}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "revoke-api-key", Method: http.MethodDelete, Path: "/api-keys/{id}",
		Summary: "Revoke an API key", Tags: tag, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{}, error) {
		if err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := s.Store.RevokeAPIKey(ctx, in.ID); err != nil {
			return nil, mapErr(err)
		}
		return &struct{}{}, nil
	})
}
