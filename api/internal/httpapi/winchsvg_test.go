package httpapi

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/ncl/mooring-api/internal/store"
)

func sampleLayout() store.Layout {
	return store.Layout{
		VesselID: "v1",
		Winches: []store.Winch{
			{
				ID: "w1", Label: "FWD-1", Station: "fwd", X: 0.3, Y: 0.4,
				Orientation: 45, DrumCount: 3, DriveType: "electric", WorstStatus: "Action",
				Drums: []store.Drum{{ID: "d1", Idx: 1, LineCount: 1}, {ID: "d2", Idx: 2, LineCount: 0}},
			},
			{
				ID: "w2", Label: "FWD-2", Station: "fwd", X: 0.6, Y: 0.4,
				Orientation: 0, DrumCount: 2, DriveType: "hydraulic", WorstStatus: "Good",
			},
			{
				ID: "w3", Label: "AFT-1", Station: "aft", X: 0.5, Y: 0.5,
				Orientation: 0, DrumCount: 4, DriveType: "electric", WorstStatus: "Monitor",
			},
		},
	}
}

func TestWinchThumbnailDataURI(t *testing.T) {
	l := sampleLayout()

	uri := winchThumbnailDataURI(l, "w1")
	const prefix = "data:image/svg+xml;base64,"
	if !strings.HasPrefix(uri, prefix) {
		t.Fatalf("missing data-uri prefix: %.40q", uri)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, prefix))
	if err != nil {
		t.Fatalf("payload is not valid base64: %v", err)
	}
	svg := string(raw)

	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatalf("not a well-formed svg element: %.60q…", svg)
	}
	// The target winch is highlighted with the accent stroke.
	if !strings.Contains(svg, colAccent) {
		t.Error("target winch should carry the accent highlight")
	}
	// Same-station winches are drawn (context), cross-station ones are not.
	if !strings.Contains(svg, "FWD-1") || !strings.Contains(svg, "FWD-2") {
		t.Error("expected both fwd winches in the thumbnail")
	}
	if strings.Contains(svg, "AFT-1") {
		t.Error("aft winch should not appear in a fwd thumbnail")
	}
	// Action status renders as a triangle (path), not a circle.
	if !strings.Contains(svg, "<path") {
		t.Error("Action status should render a triangular status mark")
	}
}

func TestWinchThumbnailUnknownID(t *testing.T) {
	if got := winchThumbnailDataURI(sampleLayout(), "nope"); got != "" {
		t.Errorf("unknown winch id should yield empty string, got %.30q", got)
	}
}
