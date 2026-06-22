package store

import "time"

// Auth --------------------------------------------------------------------

// User is an application user backed by an external OIDC identity.
type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	OIDCSub     string     `json:"sub"`
	Groups      []string   `json:"groups"`
	IsAdmin     bool       `json:"is_admin"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// AuthSession is a server-side session row (tokens stored encrypted).
type AuthSession struct {
	SID             string
	UserID          string
	AccessTokenEnc  string
	RefreshTokenEnc string
	IDTokenEnc      string
	AccessExpiresAt *time.Time
	CreatedAt       time.Time
	LastSeenAt      time.Time
}

// OIDCFlow is the short-lived state for an in-flight auth-code login.
type OIDCFlow struct {
	State        string
	CodeVerifier string
	Nonce        string
	ReturnTo     string
	CreatedAt    time.Time
}

// GroupAccess maps an OIDC group id (opaque GUID) to an access level. The
// presence of a row grants access at Level ("view"|"edit"); absence = denied.
type GroupAccess struct {
	GroupID   string    `json:"groupId"`
	Level     string    `json:"level"`
	Label     string    `json:"label"`
	UpdatedBy string    `json:"updatedBy"`
	UpdatedAt time.Time `json:"updatedAt"`
}

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
	MakerID               string   `json:"maker_id"`
	MakerName             string   `json:"maker_name"`
	LineTypeID            string   `json:"line_type_id"`
	LineTypeName          string   `json:"line_type_name"`
	ProductName           string   `json:"product_name"`
	ConstructionType      string   `json:"construction_type,omitempty"`
	DefaultLength         *float64 `json:"default_length,omitempty"`
	CanBeTurned           bool     `json:"can_be_turned"`
	ManufacturerManualRef string   `json:"manufacturer_manual_ref,omitempty"`
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
	Idx       int    `json:"idx"`
	LineCount int    `json:"line_count"`
}

type Winch struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Station     string  `json:"station"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Orientation int     `json:"orientation"`
	DrumCount   int     `json:"drum_count"`
	DriveType   string  `json:"drive_type"`
	LabelAuto   bool    `json:"label_auto"`
	WorstStatus string  `json:"worst_status,omitempty"`
	Drums       []Drum  `json:"drums"`
}

type Storage struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Station     string  `json:"station"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	LineCount   int     `json:"line_count"`
	WorstStatus string  `json:"worst_status,omitempty"`
}

type Layout struct {
	VesselID string    `json:"vessel_id"`
	Winches  []Winch   `json:"winches"`
	Storage  []Storage `json:"storage"`
}

// Lines -------------------------------------------------------------------

// LineRow is a compact row for the register table.
type LineRow struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	SerialNumber           string     `json:"serial_number"`
	TagNumber              string     `json:"tag_number,omitempty"`
	CertificateNumber      string     `json:"certificate_number,omitempty"`
	LifecycleStatus        string     `json:"lifecycle_status"`
	ProductName            string     `json:"product_name"`
	MakerName              string     `json:"maker_name"`
	LineTypeName           string     `json:"line_type_name"`
	CurrentConditionStatus string     `json:"current_condition_status,omitempty"`
	CurrentSide            string     `json:"current_side,omitempty"`
	LocationLabel          string     `json:"location_label"`
	CurrentDrumID          *string    `json:"current_drum_id,omitempty"`
	CurrentStorageID       *string    `json:"current_storage_id,omitempty"`
	Installed              bool       `json:"installed"`
	InstallAgeDays         int        `json:"install_age_days"`
	BuildAgeDays           int        `json:"build_age_days"`
	NextInspectionDue      *time.Time `json:"next_inspection_due,omitempty"`
}

// Line is the full rope record.
type Line struct {
	LineRow
	VesselID         string     `json:"vessel_id"`
	ProductID        string     `json:"product_id"`
	ConstructionType string     `json:"construction_type,omitempty"`
	Length           *float64   `json:"length,omitempty"`
	ManufactureDate  *time.Time `json:"manufacture_date,omitempty"`
	InstallationDate *time.Time `json:"installation_date,omitempty"`
	CanBeTurned      bool       `json:"can_be_turned"`
	CertificateRef   string     `json:"certificate_ref,omitempty"`

	// side tracking (live ages computed)
	SideAChangeDate *time.Time `json:"side_a_change_date,omitempty"`
	SideAAgeDays    int        `json:"side_a_age_days"`
	SideACondition  string     `json:"side_a_condition,omitempty"`
	SideBChangeDate *time.Time `json:"side_b_change_date,omitempty"`
	SideBAgeDays    int        `json:"side_b_age_days"`
	SideBCondition  string     `json:"side_b_condition,omitempty"`
	TurnDue         bool       `json:"turn_due"`

	CurrentDrumID    *string `json:"current_drum_id,omitempty"`
	CurrentStorageID *string `json:"current_storage_id,omitempty"`
	ParentLineID     *string `json:"parent_line_id,omitempty"`

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
