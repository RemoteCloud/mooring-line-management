package store

import (
	"context"
	"time"
)

// FilePhoto is a condition photo attached to a line. URL is populated by the
// handler via a presigned GET; the store leaves it blank.
type FilePhoto struct {
	ID                 string     `json:"id"`
	LineID             string     `json:"line_id"`
	InspectionID       *string    `json:"inspection_id,omitempty"`
	FileRef            string     `json:"file_ref"`
	TakenAt            *time.Time `json:"taken_at,omitempty"`
	Side               string     `json:"side,omitempty"`
	ConditionAtCapture string     `json:"condition_at_capture,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	URL                string     `json:"url,omitempty"`
}

// FileDoc is a certificate, manual or guide. URL is populated by the handler.
type FileDoc struct {
	ID          string    `json:"id"`
	LineID      *string   `json:"line_id,omitempty"`
	ProductID   *string   `json:"product_id,omitempty"`
	VesselID    *string   `json:"vessel_id,omitempty"`
	Kind        string    `json:"kind"`
	FileRef     string    `json:"file_ref"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
	URL         string    `json:"url,omitempty"`
}

// AddPhoto inserts a condition photo and emits a photo.added outbox event.
func (s *Store) AddPhoto(ctx context.Context, lineID, fileRef, side, conditionAtCapture string, takenAt *time.Time, inspectionID *string) (FilePhoto, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return FilePhoto{}, err
	}
	defer tx.Rollback(ctx)

	var vesselID string
	if err := tx.QueryRow(ctx, `SELECT vessel_id FROM mooring_line WHERE id=$1`, lineID).Scan(&vesselID); err != nil {
		return FilePhoto{}, err
	}

	id := newID()
	var p FilePhoto
	err = tx.QueryRow(ctx, `
INSERT INTO condition_photo (id, line_id, inspection_id, file_ref, taken_at, side, condition_at_capture)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id, line_id, inspection_id, file_ref, taken_at, COALESCE(side,''), COALESCE(condition_at_capture,''), created_at`,
		id, lineID, inspectionID, fileRef, takenAt, nullStr(side), nullStr(conditionAtCapture)).Scan(
		&p.ID, &p.LineID, &p.InspectionID, &p.FileRef, &p.TakenAt, &p.Side, &p.ConditionAtCapture, &p.CreatedAt)
	if err != nil {
		return FilePhoto{}, mapPgError(err)
	}

	if err := writeOutbox(ctx, tx, vesselID, "condition_photo", id, "photo.added",
		map[string]any{"id": id, "line_id": lineID, "file_ref": fileRef}); err != nil {
		return FilePhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FilePhoto{}, err
	}
	return p, nil
}

// ListPhotos returns a line's condition photos, newest first. Never nil.
func (s *Store) ListPhotos(ctx context.Context, lineID string) ([]FilePhoto, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT id, line_id, inspection_id, file_ref, taken_at, COALESCE(side,''),
       COALESCE(condition_at_capture,''), created_at
FROM condition_photo
WHERE line_id=$1
ORDER BY taken_at DESC NULLS LAST, created_at DESC`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FilePhoto{}
	for rows.Next() {
		var p FilePhoto
		if err := rows.Scan(&p.ID, &p.LineID, &p.InspectionID, &p.FileRef, &p.TakenAt,
			&p.Side, &p.ConditionAtCapture, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DeletePhoto removes a photo and returns its file_ref so the caller can purge
// the object. Propagates pgx.ErrNoRows when the photo does not exist.
func (s *Store) DeletePhoto(ctx context.Context, id string) (string, error) {
	var fileRef string
	err := s.Pool.QueryRow(ctx, `DELETE FROM condition_photo WHERE id=$1 RETURNING file_ref`, id).Scan(&fileRef)
	if err != nil {
		return "", err
	}
	return fileRef, nil
}

// AddDocument inserts a line-scoped document and emits a document.added event.
func (s *Store) AddDocument(ctx context.Context, lineID, kind, fileRef, fileName, contentType string, sizeBytes int64) (FileDoc, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return FileDoc{}, err
	}
	defer tx.Rollback(ctx)

	var vesselID string
	if err := tx.QueryRow(ctx, `SELECT vessel_id FROM mooring_line WHERE id=$1`, lineID).Scan(&vesselID); err != nil {
		return FileDoc{}, err
	}

	id := newID()
	var d FileDoc
	err = tx.QueryRow(ctx, `
INSERT INTO document (id, line_id, kind, file_ref, file_name, content_type, size_bytes)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id, line_id, product_id, vessel_id, kind, file_ref, file_name,
          COALESCE(content_type,''), size_bytes, created_at`,
		id, lineID, kind, fileRef, fileName, nullStr(contentType), sizeBytes).Scan(
		&d.ID, &d.LineID, &d.ProductID, &d.VesselID, &d.Kind, &d.FileRef, &d.FileName,
		&d.ContentType, &d.SizeBytes, &d.CreatedAt)
	if err != nil {
		return FileDoc{}, mapPgError(err)
	}

	if err := writeOutbox(ctx, tx, vesselID, "document", id, "document.added",
		map[string]any{"id": id, "line_id": lineID, "kind": kind, "file_ref": fileRef}); err != nil {
		return FileDoc{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FileDoc{}, err
	}
	return d, nil
}

// ListDocuments returns a line's documents, newest first. Never nil.
func (s *Store) ListDocuments(ctx context.Context, lineID string) ([]FileDoc, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT id, line_id, product_id, vessel_id, kind, file_ref, file_name,
       COALESCE(content_type,''), size_bytes, created_at
FROM document
WHERE line_id=$1
ORDER BY created_at DESC`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FileDoc{}
	for rows.Next() {
		var d FileDoc
		if err := rows.Scan(&d.ID, &d.LineID, &d.ProductID, &d.VesselID, &d.Kind, &d.FileRef,
			&d.FileName, &d.ContentType, &d.SizeBytes, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
