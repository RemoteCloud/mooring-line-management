package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Equipment is a received-equipment (goods-receipt) record: a piece of gear that
// arrived onboard. It is linked to a catalogue product when known (maker/model/specs
// resolve from there); otherwise maker/model are free text. Serial numbers are one or
// more, and the item may sit in a mapped storage area or a free-text location.
type Equipment struct {
	ID                string           `json:"id"`
	VesselID          string           `json:"vesselId"`
	ProductID         *string          `json:"productId,omitempty" doc:"Catalogue product this item is, when it exists in the catalogue"`
	Name              string           `json:"name" doc:"Short item name, e.g. \"Mooring tail 64mm\""`
	Maker             string           `json:"maker,omitempty" doc:"Resolved maker: the catalogue maker when linked, else the free-text maker"`
	Model             string           `json:"model,omitempty" doc:"Resolved model/article number: the catalogue model when linked, else free text"`
	ProductName       string           `json:"productName,omitempty" doc:"Catalogue product name, when linked"`
	StorageLocationID *string          `json:"storageLocationId,omitempty" doc:"Mapped storage area the item is kept in (see GET /vessels/{id}/locations?kind=storage)"`
	StorageLabel      string           `json:"storageLabel,omitempty" doc:"Free-text storage location, when not a mapped storage area"`
	Status            string           `json:"status" doc:"Lifecycle status" enum:"received,in_service,retired"`
	Notes             string           `json:"notes,omitempty"`
	ReceivedAt        *time.Time       `json:"receivedAt,omitempty" doc:"When the item was received onboard"`
	CreatedAt         time.Time        `json:"createdAt"`
	Serials           []string         `json:"serials" doc:"One or more serial numbers carried by the item"`
	Photos            []EquipmentPhoto `json:"photos,omitempty" doc:"Photos of the item; present on the single-item fetch"`
}

// EquipmentPhoto is a photo of a received item, stored in object storage. URL carries
// a freshly presigned GET link and is populated by the handler, not the database.
type EquipmentPhoto struct {
	ID          string    `json:"id"`
	EquipmentID string    `json:"equipmentId"`
	FileRef     string    `json:"fileRef" doc:"Object-storage key (internal reference)"`
	ContentType string    `json:"contentType,omitempty"`
	Caption     string    `json:"caption,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	URL         string    `json:"url,omitempty" doc:"Presigned GET URL, valid ~24h"`
}

// NewEquipmentInput carries the writable fields for create/update.
type NewEquipmentInput struct {
	ProductID         string
	Name              string
	MakerText         string
	ModelText         string
	StorageLocationID string
	StorageLabel      string
	Status            string
	Notes             string
	ReceivedAt        *time.Time
	Serials           []string
}

// equipmentSelect resolves maker/model from the catalogue product when linked, falling
// back to the row's own free-text fields, and aggregates serials into an array.
const equipmentSelect = `
SELECT e.id, e.vessel_id, e.product_id, e.name,
       COALESCE(m.name, e.maker_text, ''),
       COALESCE(p.model_number, e.model_text, ''),
       COALESCE(p.product_name, ''),
       e.storage_location_id, COALESCE(e.storage_label, ''),
       e.status, COALESCE(e.notes, ''), e.received_at, e.created_at,
       COALESCE(array_agg(es.serial_number ORDER BY es.serial_number)
                FILTER (WHERE es.serial_number IS NOT NULL), '{}')
FROM equipment e
LEFT JOIN product p ON p.id = e.product_id
LEFT JOIN maker m ON m.id = p.maker_id
LEFT JOIN equipment_serial es ON es.equipment_id = e.id`

const equipmentGroupBy = ` GROUP BY e.id, p.model_number, p.product_name, m.name`

func scanEquipment(row interface{ Scan(...any) error }) (Equipment, error) {
	var e Equipment
	err := row.Scan(&e.ID, &e.VesselID, &e.ProductID, &e.Name,
		&e.Maker, &e.Model, &e.ProductName,
		&e.StorageLocationID, &e.StorageLabel,
		&e.Status, &e.Notes, &e.ReceivedAt, &e.CreatedAt, &e.Serials)
	return e, err
}

const equipmentPhotoSelect = `
SELECT id, equipment_id, file_ref, COALESCE(content_type, ''), COALESCE(caption, ''), created_at
FROM equipment_photo`

func scanEquipmentPhoto(row interface{ Scan(...any) error }) (EquipmentPhoto, error) {
	var p EquipmentPhoto
	err := row.Scan(&p.ID, &p.EquipmentID, &p.FileRef, &p.ContentType, &p.Caption, &p.CreatedAt)
	return p, err
}

// dedupeNonEmpty trims, drops blanks, and removes duplicate serials so a repeated
// value never trips the (equipment_id, serial_number) unique index.
func dedupeNonEmpty(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// CreateEquipment records a received item with its serials and emits equipment.received.
func (s *Store) CreateEquipment(ctx context.Context, vesselID string, in NewEquipmentInput) (Equipment, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Equipment{}, err
	}
	defer tx.Rollback(ctx)

	id := newID()
	status := in.Status
	if status == "" {
		status = "received"
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO equipment (id, vessel_id, product_id, name, maker_text, model_text,
                       storage_location_id, storage_label, status, notes, received_at)
VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,NULLIF($8,''),$9,NULLIF($10,''),$11)`,
		id, vesselID, nullUUID(in.ProductID), in.Name, in.MakerText, in.ModelText,
		nullUUID(in.StorageLocationID), in.StorageLabel, status, in.Notes, in.ReceivedAt); err != nil {
		return Equipment{}, mapPgError(err)
	}

	serials := dedupeNonEmpty(in.Serials)
	for _, sn := range serials {
		if _, err := tx.Exec(ctx,
			`INSERT INTO equipment_serial (id, equipment_id, serial_number) VALUES ($1,$2,$3)`,
			newID(), id, sn); err != nil {
			return Equipment{}, mapPgError(err)
		}
	}

	if err := writeOutbox(ctx, tx, vesselID, "equipment", id, "equipment.received",
		map[string]any{"id": id, "name": in.Name, "serials": serials}); err != nil {
		return Equipment{}, err
	}

	e, err := scanEquipment(tx.QueryRow(ctx, equipmentSelect+` WHERE e.id=$1`+equipmentGroupBy, id))
	if err != nil {
		return Equipment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Equipment{}, err
	}
	return e, nil
}

// ListEquipment returns the vessel's received equipment, newest first, optionally
// filtered by catalogue product and/or status. Photos are omitted from the list.
func (s *Store) ListEquipment(ctx context.Context, vesselID, productID, status string) ([]Equipment, error) {
	rows, err := s.Pool.Query(ctx, equipmentSelect+`
WHERE e.vessel_id=$1
  AND ($2::uuid IS NULL OR e.product_id=$2)
  AND ($3::text IS NULL OR e.status=$3)`+equipmentGroupBy+`
ORDER BY e.created_at DESC`,
		vesselID, nullUUID(productID), nullStr(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Equipment{}
	for rows.Next() {
		e, err := scanEquipment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetEquipment returns one item with its serials and photos. Unknown id → pgx.ErrNoRows.
func (s *Store) GetEquipment(ctx context.Context, id string) (Equipment, error) {
	e, err := scanEquipment(s.Pool.QueryRow(ctx, equipmentSelect+` WHERE e.id=$1`+equipmentGroupBy, id))
	if err != nil {
		return Equipment{}, err
	}
	photos, err := s.ListEquipmentPhotos(ctx, id)
	if err != nil {
		return Equipment{}, err
	}
	e.Photos = photos
	return e, nil
}

// UpdateEquipment replaces the item's mutable fields and its serial set. Unknown id →
// pgx.ErrNoRows.
func (s *Store) UpdateEquipment(ctx context.Context, id string, in NewEquipmentInput) (Equipment, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Equipment{}, err
	}
	defer tx.Rollback(ctx)

	status := in.Status
	if status == "" {
		status = "received"
	}
	tag, err := tx.Exec(ctx, `
UPDATE equipment SET product_id=$2, name=$3, maker_text=NULLIF($4,''), model_text=NULLIF($5,''),
                     storage_location_id=$6, storage_label=NULLIF($7,''), status=$8,
                     notes=NULLIF($9,''), received_at=$10
WHERE id=$1`,
		id, nullUUID(in.ProductID), in.Name, in.MakerText, in.ModelText,
		nullUUID(in.StorageLocationID), in.StorageLabel, status, in.Notes, in.ReceivedAt)
	if err != nil {
		return Equipment{}, mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return Equipment{}, pgx.ErrNoRows
	}

	if _, err := tx.Exec(ctx, `DELETE FROM equipment_serial WHERE equipment_id=$1`, id); err != nil {
		return Equipment{}, err
	}
	for _, sn := range dedupeNonEmpty(in.Serials) {
		if _, err := tx.Exec(ctx,
			`INSERT INTO equipment_serial (id, equipment_id, serial_number) VALUES ($1,$2,$3)`,
			newID(), id, sn); err != nil {
			return Equipment{}, mapPgError(err)
		}
	}

	e, err := scanEquipment(tx.QueryRow(ctx, equipmentSelect+` WHERE e.id=$1`+equipmentGroupBy, id))
	if err != nil {
		return Equipment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Equipment{}, err
	}
	return e, nil
}

// DeleteEquipment removes an item (serials + photo rows cascade). Unknown id →
// pgx.ErrNoRows. Photo objects in storage are left as harmless orphans.
func (s *Store) DeleteEquipment(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM equipment WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// AddEquipmentPhoto records a photo (already uploaded to object storage) and emits
// equipment.photo.added. Unknown equipment id → pgx.ErrNoRows.
func (s *Store) AddEquipmentPhoto(ctx context.Context, equipmentID, fileRef, contentType, caption string) (EquipmentPhoto, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return EquipmentPhoto{}, err
	}
	defer tx.Rollback(ctx)

	var vesselID string
	if err := tx.QueryRow(ctx, `SELECT vessel_id FROM equipment WHERE id=$1`, equipmentID).Scan(&vesselID); err != nil {
		// pgx.ErrNoRows propagates so the handler returns 404.
		return EquipmentPhoto{}, err
	}

	id := newID()
	p, err := scanEquipmentPhoto(tx.QueryRow(ctx, `
INSERT INTO equipment_photo (id, equipment_id, file_ref, content_type, caption)
VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''))
RETURNING id, equipment_id, file_ref, COALESCE(content_type,''), COALESCE(caption,''), created_at`,
		id, equipmentID, fileRef, contentType, caption))
	if err != nil {
		return EquipmentPhoto{}, err
	}

	if err := writeOutbox(ctx, tx, vesselID, "equipment", id, "equipment.photo.added",
		map[string]any{"id": id, "equipmentId": equipmentID}); err != nil {
		return EquipmentPhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EquipmentPhoto{}, err
	}
	return p, nil
}

// ListEquipmentPhotos returns an item's photos, oldest first.
func (s *Store) ListEquipmentPhotos(ctx context.Context, equipmentID string) ([]EquipmentPhoto, error) {
	rows, err := s.Pool.Query(ctx, equipmentPhotoSelect+` WHERE equipment_id=$1 ORDER BY created_at`, equipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EquipmentPhoto{}
	for rows.Next() {
		p, err := scanEquipmentPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
