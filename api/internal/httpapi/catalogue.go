package httpapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/ncl/mooring-api/internal/store"
)

func registerCatalogue(api huma.API, s *Server) {
	tag := []string{"catalogue"}

	huma.Register(api, huma.Operation{
		OperationID: "list-makers", Method: http.MethodGet, Path: "/makers",
		Summary: "List makers", Description: "Rope manufacturers in the catalogue.", Tags: tag,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.Maker }, error) {
		m, err := s.Store.ListMakers(ctx)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.Maker }{Body: m}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-maker", Method: http.MethodPost, Path: "/makers",
		Summary: "Create maker", Tags: tag, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		Body struct {
			Name  string `json:"name" minLength:"1"`
			Notes string `json:"notes,omitempty"`
		}
	}) (*struct{ Body store.Maker }, error) {
		m, err := s.Store.CreateMaker(ctx, in.Body.Name, in.Body.Notes)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Maker }{Body: m}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-maker", Method: http.MethodPatch, Path: "/makers/{id}",
		Summary: "Update maker", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			Name  string `json:"name" minLength:"1"`
			Notes string `json:"notes,omitempty"`
		}
	}) (*struct{ Body store.Maker }, error) {
		m, err := s.Store.UpdateMaker(ctx, in.ID, in.Body.Name, in.Body.Notes)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Maker }{Body: m}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-line-types", Method: http.MethodGet, Path: "/line-types",
		Summary: "List line types", Description: "Functional line categories (e.g. mooring tail, main line, lashing).", Tags: tag,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []store.LineType }, error) {
		t, err := s.Store.ListLineTypes(ctx)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.LineType }{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-line-type", Method: http.MethodPost, Path: "/line-types",
		Summary: "Create line type", Tags: tag, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		Body struct {
			Name        string `json:"name" minLength:"1"`
			Description string `json:"description,omitempty"`
		}
	}) (*struct{ Body store.LineType }, error) {
		t, err := s.Store.CreateLineType(ctx, in.Body.Name, in.Body.Description)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.LineType }{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-line-type", Method: http.MethodPatch, Path: "/line-types/{id}",
		Summary: "Update line type", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			Name        string `json:"name" minLength:"1"`
			Description string `json:"description,omitempty"`
		}
	}) (*struct{ Body store.LineType }, error) {
		t, err := s.Store.UpdateLineType(ctx, in.ID, in.Body.Name, in.Body.Description)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.LineType }{Body: t}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-products", Method: http.MethodGet, Path: "/products",
		Summary: "List products", Description: "Catalogue products (a maker + line type with specs). Lines are registered from these.", Tags: tag,
	}, func(ctx context.Context, in *struct {
		MakerID    string `query:"makerId"`
		LineTypeID string `query:"lineTypeId"`
	}) (*struct{ Body []store.Product }, error) {
		p, err := s.Store.ListProducts(ctx, in.MakerID, in.LineTypeID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.Product }{Body: p}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-product", Method: http.MethodGet, Path: "/products/{id}",
		Summary: "Get a product", Description: "Returns one catalogue product with its full specs.", Tags: tag, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body store.Product }, error) {
		p, err := s.Store.GetProduct(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Product }{Body: p}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-product", Method: http.MethodPost, Path: "/products",
		Summary: "Create product", Tags: tag, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		Body struct {
			MakerID          string   `json:"makerId" format:"uuid"`
			LineTypeID       string   `json:"lineTypeId" format:"uuid"`
			ProductName      string   `json:"productName" minLength:"1"`
			ModelNumber      string   `json:"modelNumber,omitempty"`
			ConstructionType string   `json:"constructionType,omitempty"`
			DefaultLength    *float64 `json:"defaultLength,omitempty"`
			SWL              *float64 `json:"swl,omitempty"`
			BreakLoad        *float64 `json:"breakLoad,omitempty"`
			CanBeTurned      bool     `json:"canBeTurned"`
			ManualRef        string   `json:"manufacturerManualRef,omitempty"`
			Notes            string   `json:"notes,omitempty"`
		}
	}) (*struct{ Body store.Product }, error) {
		p, err := s.Store.CreateProduct(ctx, store.NewProductInput{
			MakerID: in.Body.MakerID, LineTypeID: in.Body.LineTypeID,
			ProductName: in.Body.ProductName, ModelNumber: in.Body.ModelNumber,
			ConstructionType: in.Body.ConstructionType,
			DefaultLength: in.Body.DefaultLength, SWL: in.Body.SWL, BreakLoad: in.Body.BreakLoad,
			CanBeTurned: in.Body.CanBeTurned,
			ManualRef: in.Body.ManualRef, Notes: in.Body.Notes,
		})
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Product }{Body: p}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-product", Method: http.MethodPatch, Path: "/products/{id}",
		Summary: "Update product", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			MakerID          string   `json:"makerId" format:"uuid"`
			LineTypeID       string   `json:"lineTypeId" format:"uuid"`
			ProductName      string   `json:"productName" minLength:"1"`
			ModelNumber      string   `json:"modelNumber,omitempty"`
			ConstructionType string   `json:"constructionType,omitempty"`
			DefaultLength    *float64 `json:"defaultLength,omitempty"`
			SWL              *float64 `json:"swl,omitempty"`
			BreakLoad        *float64 `json:"breakLoad,omitempty"`
			CanBeTurned      bool     `json:"canBeTurned"`
			ManualRef        string   `json:"manufacturerManualRef,omitempty"`
			Notes            string   `json:"notes,omitempty"`
		}
	}) (*struct{ Body store.Product }, error) {
		p, err := s.Store.UpdateProduct(ctx, in.ID, store.NewProductInput{
			MakerID: in.Body.MakerID, LineTypeID: in.Body.LineTypeID,
			ProductName: in.Body.ProductName, ModelNumber: in.Body.ModelNumber,
			ConstructionType: in.Body.ConstructionType,
			DefaultLength: in.Body.DefaultLength, SWL: in.Body.SWL, BreakLoad: in.Body.BreakLoad,
			CanBeTurned: in.Body.CanBeTurned,
			ManualRef: in.Body.ManualRef, Notes: in.Body.Notes,
		})
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Product }{Body: p}, nil
	})
}
