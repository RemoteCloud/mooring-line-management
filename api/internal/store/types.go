package store

import "time"

// Catalogue ---------------------------------------------------------------

type Maker struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Notes string `json:"notes,omitempty"`
}

type LineType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Product struct {
	ID                    string   `json:"id"`
	MakerID               string   `json:"makerId"`
	MakerName             string   `json:"makerName"`
	LineTypeID            string   `json:"lineTypeId"`
	LineTypeName          string   `json:"lineTypeName"`
	ProductName           string   `json:"productName"`
	ModelNumber           string   `json:"modelNumber,omitempty" doc:"Manufacturer model/article number"`
	ConstructionType      string   `json:"constructionType,omitempty" doc:"Rope construction, e.g. 8-strand, 12-strand, double-braid"`
	DefaultLength         *float64 `json:"defaultLength,omitempty" doc:"Default line length in metres"`
	SWL                   *float64 `json:"swl,omitempty" doc:"Safe working load in tonnes"`
	BreakLoad             *float64 `json:"breakLoad,omitempty" doc:"Break load / minimum breaking load (MBL) in tonnes"`
	CanBeTurned           bool     `json:"canBeTurned" doc:"Whether lines of this product can be turned end-for-end to even out wear"`
	ManufacturerManualRef string   `json:"manufacturerManualRef,omitempty" doc:"Reference to the manufacturer's manual/datasheet"`
	Notes                 string   `json:"notes,omitempty"`
}

// Vessel + layout ---------------------------------------------------------

type Vessel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IMO  string `json:"imo,omitempty"`
}

type Drum struct {
	ID        string `json:"id"`
	Idx       int    `json:"idx" doc:"1-based drum position on the winch"`
	LineCount int    `json:"lineCount" doc:"Number of lines currently on this drum"`
}

type Winch struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Station     string  `json:"station" doc:"Which deck the winch is on" enum:"fwd,aft"`
	X           float64 `json:"x" doc:"Normalized horizontal position (0..1) across the deck"`
	Y           float64 `json:"y" doc:"Normalized vertical position (0..1) down the deck"`
	Orientation int     `json:"orientation" doc:"Rotation in degrees" enum:"0,45,-45,90,-90"`
	DrumCount   int      `json:"drumCount" doc:"Number of drums on the winch"`
	DriveType   string   `json:"driveType" doc:"Winch drive" enum:"electric,hydraulic"`
	LabelAuto   bool     `json:"labelAuto" doc:"Whether the label is auto-generated from position"`
	SWL         *float64 `json:"swl,omitempty" doc:"Safe working load in tonnes"`
	BreakLoad   *float64 `json:"breakLoad,omitempty" doc:"Break load in tonnes"`
	WorstStatus string   `json:"worstStatus,omitempty" doc:"Worst condition among the lines on this winch" enum:"Good,Monitor,Action"`
	Drums       []Drum   `json:"drums"`
	// Thumbnail is a base64 SVG data-URI of this winch highlighted on the deck.
	// Only populated when the layout is requested with ?thumbnails=true.
	Thumbnail string `json:"thumbnail,omitempty"`
}

type Storage struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Station     string  `json:"station"`
	OnMap       bool    `json:"onMap"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	LineCount   int     `json:"lineCount"`
	WorstStatus string  `json:"worstStatus,omitempty"`
}

type Layout struct {
	VesselID string    `json:"vesselId"`
	Winches  []Winch   `json:"winches"`
	Storage  []Storage `json:"storage"`
}

// Lines -------------------------------------------------------------------

// LineRow is a compact row for the register table.
type LineRow struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	SerialNumber           string     `json:"serialNumber"`
	TagNumber              string     `json:"tagNumber,omitempty"`
	CertificateNumber      string     `json:"certificateNumber,omitempty"`
	LifecycleStatus        string     `json:"lifecycleStatus"`
	ProductName            string     `json:"productName"`
	MakerName              string     `json:"makerName"`
	LineTypeName           string     `json:"lineTypeName"`
	CurrentConditionStatus string     `json:"currentConditionStatus,omitempty"`
	CurrentSide            string     `json:"currentSide,omitempty"`
	LocationLabel          string     `json:"locationLabel"`
	CurrentDrumID          *string    `json:"currentDrumId,omitempty"`
	CurrentStorageID       *string    `json:"currentStorageId,omitempty"`
	Installed              bool       `json:"installed"`
	InstallAgeDays         int        `json:"installAgeDays"`
	BuildAgeDays           int        `json:"buildAgeDays"`
	NextInspectionDue      *time.Time `json:"nextInspectionDue,omitempty"`
}

// Line is the full rope record.
type Line struct {
	LineRow
	VesselID         string     `json:"vesselId"`
	ProductID        string     `json:"productId"`
	ConstructionType string     `json:"constructionType,omitempty"`
	SWL              *float64   `json:"swl,omitempty"`       // inherited from the product, tonnes
	BreakLoad        *float64   `json:"breakLoad,omitempty"` // inherited from the product, tonnes
	Length           *float64   `json:"length,omitempty"`
	ManufactureDate  *time.Time `json:"manufactureDate,omitempty"`
	InstallationDate *time.Time `json:"installationDate,omitempty"`
	CanBeTurned      bool       `json:"canBeTurned"`
	CertificateRef   string     `json:"certificateRef,omitempty"`

	// side tracking (live ages computed)
	SideAChangeDate *time.Time `json:"sideAChangeDate,omitempty"`
	SideAAgeDays    int        `json:"sideAAgeDays"`
	SideACondition  string     `json:"sideACondition,omitempty"`
	SideBChangeDate *time.Time `json:"sideBChangeDate,omitempty"`
	SideBAgeDays    int        `json:"sideBAgeDays"`
	SideBCondition  string     `json:"sideBCondition,omitempty"`
	TurnDue         bool       `json:"turnDue"`

	CurrentDrumID    *string `json:"currentDrumId,omitempty"`
	CurrentStorageID *string `json:"currentStorageId,omitempty"`
	ParentLineID     *string `json:"parentLineId,omitempty"`

	Components []Line `json:"components"`
}

// NewLineInput carries the fields needed to register a line or component.
type NewLineInput struct {
	ProductID         string
	Name              string
	SerialNumber      string
	TagNumber         string
	CertificateNumber string
	LifecycleStatus   string
	Length            *float64
	ManufactureDate   *time.Time
	InstallationDate  *time.Time
	CurrentSide       string
	ParentLineID      *string
}
