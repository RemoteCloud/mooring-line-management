package httpapi

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/ncl/mooring-api/internal/storage"
	"github.com/ncl/mooring-api/internal/store"
)

// equipmentBody is the create/update request shape for a received-equipment record.
type equipmentBody struct {
	ProductID         string   `json:"productId,omitempty" format:"uuid" doc:"Catalogue product id when the item exists in the catalogue (maker/model/specs resolve from it)"`
	Name              string   `json:"name" minLength:"1" doc:"Short item name" example:"Mooring tail 64mm"`
	MakerText         string   `json:"makerText,omitempty" doc:"Maker name when the item is not in the catalogue" example:"Lankhorst"`
	ModelText         string   `json:"modelText,omitempty" doc:"Model/article number when not in the catalogue" example:"Lankoforce-88"`
	StorageLocationID string   `json:"storageLocationId,omitempty" format:"uuid" doc:"Mapped storage area id (see GET /vessels/{vesselId}/locations?kind=storage)"`
	StorageLabel      string   `json:"storageLabel,omitempty" doc:"Free-text storage location when not a mapped area" example:"Bosun store, shelf 3"`
	Status            string   `json:"status,omitempty" enum:"received,in_service,retired" doc:"Lifecycle status; defaults to received"`
	Notes             string   `json:"notes,omitempty"`
	ReceivedAt        string   `json:"receivedAt,omitempty" format:"date" doc:"Date the item was received onboard (YYYY-MM-DD)"`
	Serials           []string `json:"serials,omitempty" doc:"One or more serial numbers carried by the item" example:"[\"AB-1024\",\"AB-1024-M\"]"`
}

func (b equipmentBody) toInput() store.NewEquipmentInput {
	return store.NewEquipmentInput{
		ProductID:         b.ProductID,
		Name:              b.Name,
		MakerText:         b.MakerText,
		ModelText:         b.ModelText,
		StorageLocationID: b.StorageLocationID,
		StorageLabel:      b.StorageLabel,
		Status:            b.Status,
		Notes:             b.Notes,
		ReceivedAt:        parseDate(b.ReceivedAt),
		Serials:           b.Serials,
	}
}

func registerEquipment(api huma.API, s *Server) {
	tag := []string{"equipment"}

	fs, ferr := storage.New(s.Cfg)
	if ferr == nil {
		_ = fs.EnsureBucket(context.Background())
	}

	huma.Register(api, huma.Operation{
		OperationID: "equipment-create", Method: http.MethodPost, Path: "/vessels/{vesselId}/equipment",
		Summary: "Register received equipment",
		Description: "Records a piece of gear that arrived onboard (a mooring line, shackle, spare, …). " +
			"Link it to a catalogue product via `productId` to inherit maker/model/specs, or pass " +
			"`makerText`/`modelText` for items not in the catalogue. Provide one or more `serials`, and " +
			"either a mapped `storageLocationId` or a free-text `storageLabel`. Fires `equipment.received`.",
		Tags: tag, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		VesselID string `path:"vesselId" format:"uuid"`
		Body     equipmentBody
	}) (*struct{ Body store.Equipment }, error) {
		e, err := s.Store.CreateEquipment(ctx, s.vessel(in.VesselID), in.Body.toInput())
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Equipment }{Body: e}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "equipment-list", Method: http.MethodGet, Path: "/vessels/{vesselId}/equipment",
		Summary:     "List received equipment",
		Description: "Returns the vessel's received equipment, newest first. Filter by catalogue `productId` and/or `status`. Photos are omitted here; fetch a single item for its photos.",
		Tags:        tag,
	}, func(ctx context.Context, in *struct {
		VesselID  string `path:"vesselId" format:"uuid"`
		ProductID string `query:"productId" doc:"Only items linked to this catalogue product"`
		Status    string `query:"status" enum:"received,in_service,retired" doc:"Only items in this lifecycle status"`
	}) (*struct{ Body []store.Equipment }, error) {
		list, err := s.Store.ListEquipment(ctx, s.vessel(in.VesselID), in.ProductID, in.Status)
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body []store.Equipment }{Body: list}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "equipment-get", Method: http.MethodGet, Path: "/equipment/{id}",
		Summary:     "Get a received-equipment item",
		Description: "Returns one item with its serial numbers and photos. Each photo carries a freshly presigned `url`.",
		Tags:        tag, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body store.Equipment }, error) {
		e, err := s.Store.GetEquipment(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		if fs != nil {
			for i := range e.Photos {
				if url, perr := fs.PresignGet(ctx, e.Photos[i].FileRef); perr == nil {
					e.Photos[i].URL = url
				}
			}
		}
		return &struct{ Body store.Equipment }{Body: e}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "equipment-update", Method: http.MethodPatch, Path: "/equipment/{id}",
		Summary:     "Update a received-equipment item",
		Description: "Replaces the item's fields and its full set of serial numbers.",
		Tags:        tag, Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body equipmentBody
	}) (*struct{ Body store.Equipment }, error) {
		e, err := s.Store.UpdateEquipment(ctx, in.ID, in.Body.toInput())
		if err != nil {
			return nil, mapErr(err)
		}
		return &struct{ Body store.Equipment }{Body: e}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "equipment-delete", Method: http.MethodDelete, Path: "/equipment/{id}",
		Summary: "Delete a received-equipment item", Tags: tag,
		DefaultStatus: http.StatusNoContent, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{}, error) {
		if err := s.Store.DeleteEquipment(ctx, in.ID); err != nil {
			return nil, mapErr(err)
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "equipment-add-photo", Method: http.MethodPost, Path: "/equipment/{id}/photos",
		Summary: "Upload a photo of a received item",
		Description: "Attaches a photo to the item. The image is sent base64-encoded in `fileBase64` and stored " +
			"in object storage; reads return a presigned URL. Fires `equipment.photo.added`.",
		Tags: tag, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			FileBase64  string `json:"fileBase64" doc:"The image bytes, base64-encoded"`
			ContentType string `json:"contentType,omitempty" example:"image/jpeg"`
			Caption     string `json:"caption,omitempty" doc:"Optional caption, e.g. \"label on reel\""`
		}
	}) (*struct{ Body store.EquipmentPhoto }, error) {
		if fs == nil {
			return nil, huma.Error500InternalServerError("object storage unavailable")
		}
		data, err := base64.StdEncoding.DecodeString(in.Body.FileBase64)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid base64 file")
		}
		key := "equipment/" + in.ID + "/" + uuid.NewString()
		if err := fs.Put(ctx, key, data, in.Body.ContentType); err != nil {
			return nil, huma.Error500InternalServerError("upload failed", err)
		}
		p, err := s.Store.AddEquipmentPhoto(ctx, in.ID, key, in.Body.ContentType, in.Body.Caption)
		if err != nil {
			return nil, mapErr(err)
		}
		if url, perr := fs.PresignGet(ctx, p.FileRef); perr == nil {
			p.URL = url
		}
		return &struct{ Body store.EquipmentPhoto }{Body: p}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "equipment-list-photos", Method: http.MethodGet, Path: "/equipment/{id}/photos",
		Summary: "List photos of a received item", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body []store.EquipmentPhoto }, error) {
		photos, err := s.Store.ListEquipmentPhotos(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		if fs != nil {
			for i := range photos {
				if url, perr := fs.PresignGet(ctx, photos[i].FileRef); perr == nil {
					photos[i].URL = url
				}
			}
		}
		return &struct{ Body []store.EquipmentPhoto }{Body: photos}, nil
	})
}
