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

// registerFiles wires the photos + certificates/documents routes. Uploads are
// base64-encoded JSON bodies (simpler and more robust than multipart). Reads
// always carry a freshly presigned URL.
//
// NOTE: this is defined but intentionally not called here — the orchestrator
// wires it into NewAPI and removes the list-line-files stub from lines.go.
func registerFiles(api huma.API, s *Server) {
	tag := []string{"files"}

	fs, ferr := storage.New(s.Cfg)
	if ferr == nil {
		_ = fs.EnsureBucket(context.Background())
	}

	huma.Register(api, huma.Operation{
		OperationID: "file-add-photo", Method: http.MethodPost, Path: "/lines/{id}/photos",
		Summary: "Upload a condition photo for a line", Tags: tag,
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			FileBase64         string `json:"fileBase64"`
			ContentType        string `json:"contentType,omitempty"`
			TakenAt            string `json:"takenAt,omitempty" format:"date"`
			Side               string `json:"side,omitempty" enum:"A,B,n/a"`
			ConditionAtCapture string `json:"conditionAtCapture,omitempty" enum:"Good,Monitor,Action"`
			InspectionID       string `json:"inspectionId,omitempty"`
		}
	}) (*struct{ Body store.FilePhoto }, error) {
		if fs == nil {
			return nil, huma.Error500InternalServerError("object storage unavailable")
		}
		data, err := base64.StdEncoding.DecodeString(in.Body.FileBase64)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid base64 file")
		}
		key := "photos/" + in.ID + "/" + uuid.NewString()
		if err := fs.Put(ctx, key, data, in.Body.ContentType); err != nil {
			return nil, huma.Error500InternalServerError("upload failed", err)
		}
		var inspID *string
		if in.Body.InspectionID != "" {
			inspID = &in.Body.InspectionID
		}
		p, err := s.Store.AddPhoto(ctx, in.ID, key, in.Body.Side, in.Body.ConditionAtCapture,
			parseDate(in.Body.TakenAt), inspID)
		if err != nil {
			return nil, mapErr(err)
		}
		if url, perr := fs.PresignGet(ctx, p.FileRef); perr == nil {
			p.URL = url
		}
		return &struct{ Body store.FilePhoto }{Body: p}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "file-list-photos", Method: http.MethodGet, Path: "/lines/{id}/photos",
		Summary: "List condition photos for a line", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body []store.FilePhoto }, error) {
		if fs == nil {
			return nil, huma.Error500InternalServerError("object storage unavailable")
		}
		photos, err := s.Store.ListPhotos(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		for i := range photos {
			if url, perr := fs.PresignGet(ctx, photos[i].FileRef); perr == nil {
				photos[i].URL = url
			}
		}
		return &struct{ Body []store.FilePhoto }{Body: photos}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "file-del-photo", Method: http.MethodDelete, Path: "/photos/{id}",
		Summary: "Delete a condition photo", Tags: tag,
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{}, error) {
		if fs == nil {
			return nil, huma.Error500InternalServerError("object storage unavailable")
		}
		fileRef, err := s.Store.DeletePhoto(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		_ = fs.Remove(ctx, fileRef)
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "file-add-cert", Method: http.MethodPost, Path: "/lines/{id}/certificate",
		Summary: "Upload a certificate, manual or guide for a line", Tags: tag,
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id" format:"uuid"`
		Body struct {
			FileBase64  string `json:"fileBase64"`
			FileName    string `json:"fileName"`
			ContentType string `json:"contentType,omitempty"`
			Kind        string `json:"kind,omitempty" enum:"certificate,manual,guide,delivery"`
		}
	}) (*struct{ Body store.FileDoc }, error) {
		if fs == nil {
			return nil, huma.Error500InternalServerError("object storage unavailable")
		}
		data, err := base64.StdEncoding.DecodeString(in.Body.FileBase64)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid base64 file")
		}
		kind := in.Body.Kind
		if kind == "" {
			kind = "certificate"
		}
		key := "docs/" + in.ID + "/" + uuid.NewString() + "-" + in.Body.FileName
		if err := fs.Put(ctx, key, data, in.Body.ContentType); err != nil {
			return nil, huma.Error500InternalServerError("upload failed", err)
		}
		d, err := s.Store.AddDocument(ctx, in.ID, kind, key, in.Body.FileName, in.Body.ContentType, int64(len(data)))
		if err != nil {
			return nil, mapErr(err)
		}
		if url, perr := fs.PresignGet(ctx, d.FileRef); perr == nil {
			d.URL = url
		}
		return &struct{ Body store.FileDoc }{Body: d}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "file-list", Method: http.MethodGet, Path: "/lines/{id}/files",
		Summary: "List documents (certificates/manuals/guides) for a line", Tags: tag,
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" format:"uuid"`
	}) (*struct{ Body []store.FileDoc }, error) {
		if fs == nil {
			return nil, huma.Error500InternalServerError("object storage unavailable")
		}
		docs, err := s.Store.ListDocuments(ctx, in.ID)
		if err != nil {
			return nil, mapErr(err)
		}
		for i := range docs {
			if url, perr := fs.PresignGet(ctx, docs[i].FileRef); perr == nil {
				docs[i].URL = url
			}
		}
		return &struct{ Body []store.FileDoc }{Body: docs}, nil
	})
}
