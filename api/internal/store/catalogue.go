package store

import (
	"context"

	"github.com/google/uuid"
)

func newID() string { return uuid.Must(uuid.NewV7()).String() }

func (s *Store) ListMakers(ctx context.Context) ([]Maker, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, COALESCE(notes,'') FROM maker ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Maker
	for rows.Next() {
		var m Maker
		if err := rows.Scan(&m.ID, &m.Name, &m.Notes); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CreateMaker(ctx context.Context, name, notes string) (Maker, error) {
	m := Maker{ID: newID(), Name: name, Notes: notes}
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO maker (id, name, notes) VALUES ($1,$2,NULLIF($3,''))`, m.ID, name, notes)
	return m, err
}

func (s *Store) ListLineTypes(ctx context.Context) ([]LineType, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, COALESCE(description,'') FROM line_type ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LineType
	for rows.Next() {
		var t LineType
		if err := rows.Scan(&t.ID, &t.Name, &t.Description); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateLineType(ctx context.Context, name, desc string) (LineType, error) {
	t := LineType{ID: newID(), Name: name, Description: desc}
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO line_type (id, name, description) VALUES ($1,$2,NULLIF($3,''))`, t.ID, name, desc)
	return t, err
}

const productSelect = `
SELECT p.id, p.maker_id, m.name, p.line_type_id, lt.name,
       p.product_name, COALESCE(p.construction_type,''), p.default_length,
       p.can_be_turned, COALESCE(p.manufacturer_manual_ref,''), COALESCE(p.notes,'')
FROM product p
JOIN maker m ON m.id = p.maker_id
JOIN line_type lt ON lt.id = p.line_type_id`

func scanProduct(row interface{ Scan(...any) error }) (Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.MakerID, &p.MakerName, &p.LineTypeID, &p.LineTypeName,
		&p.ProductName, &p.ConstructionType, &p.DefaultLength,
		&p.CanBeTurned, &p.ManufacturerManualRef, &p.Notes)
	return p, err
}

func (s *Store) ListProducts(ctx context.Context, makerID, lineTypeID string) ([]Product, error) {
	rows, err := s.Pool.Query(ctx, productSelect+`
WHERE ($1::uuid IS NULL OR p.maker_id = $1)
  AND ($2::uuid IS NULL OR p.line_type_id = $2)
ORDER BY m.name, p.product_name`,
		nullUUID(makerID), nullUUID(lineTypeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProduct(ctx context.Context, id string) (Product, error) {
	return scanProduct(s.Pool.QueryRow(ctx, productSelect+` WHERE p.id = $1`, id))
}

type NewProductInput struct {
	MakerID, LineTypeID, ProductName, ConstructionType, ManualRef, Notes string
	DefaultLength                                                        *float64
	CanBeTurned                                                          bool
}

func (s *Store) CreateProduct(ctx context.Context, in NewProductInput) (Product, error) {
	id := newID()
	_, err := s.Pool.Exec(ctx, `
INSERT INTO product (id, maker_id, line_type_id, product_name, construction_type,
                     default_length, can_be_turned, manufacturer_manual_ref, notes)
VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),NULLIF($9,''))`,
		id, in.MakerID, in.LineTypeID, in.ProductName, in.ConstructionType,
		in.DefaultLength, in.CanBeTurned, in.ManualRef, in.Notes)
	if err != nil {
		return Product{}, err
	}
	return s.GetProduct(ctx, id)
}

// nullUUID turns an empty string into a nil so the SQL `$n::uuid IS NULL` checks work.
func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}
