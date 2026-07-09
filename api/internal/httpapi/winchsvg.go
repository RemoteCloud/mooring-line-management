package httpapi

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"

	"github.com/ncl/mooring-api/internal/store"
)

// Per-winch deck thumbnail. This is a faithful Go port of the React deck symbols
// (web/src/features/deck/symbols.tsx): same 1000x600 viewBox, same geometry, with
// the dark-theme colors from web/src/styles.css inlined (a data-URI SVG has no
// stylesheet). It draws the winch's station — hull + every winch on that station —
// with the target winch highlighted, so an external consumer gets a visual of
// where the winch sits without re-rendering the whole map.

const (
	vbW     = 1000.0
	vbH     = 600.0
	drumW   = 28.0
	drumH   = 42.0
	drumGap = 5.0
	pad     = 11.0
)

// Dark-theme palette (mirrors :root in styles.css).
const (
	colBg         = "#0e1726"
	colWinchFill  = "#1c2c46"
	colBorder     = "#26395a"
	colText       = "#e6edf6"
	colMuted      = "#93a4bf"
	colAccent     = "#2f9bdb"
	colAccentDeep = "#0b3d5c"
	colGood       = "#2faa6b"
	colMonitor    = "#e0a325"
	colAction     = "#e05a4d"
)

func condClass(status string) string {
	switch status {
	case "Good":
		return "good"
	case "Monitor":
		return "monitor"
	case "Action":
		return "action"
	default:
		return ""
	}
}

func statusColor(status string) string {
	switch condClass(status) {
	case "good":
		return colGood
	case "monitor":
		return colMonitor
	case "action":
		return colAction
	default:
		return colBorder
	}
}

func winchBoxDims(drumCount int) (bw, bh, inner float64) {
	n := float64(drumCount)
	inner = n*drumW + (n-1)*drumGap
	return inner + pad*2, drumH + pad*2, inner
}

func hullPoints(station string) string {
	if station == "fwd" {
		return fmt.Sprintf("%g,20 850,170 850,560 150,560 150,170", vbW/2)
	}
	return "150,40 850,40 850,470 640,580 360,580 150,470"
}

// winchThumbnailDataURI returns a base64 SVG data-URI of the given winch highlighted
// on its station's deck, or "" if the winch is not in the layout.
func winchThumbnailDataURI(l store.Layout, winchID string) string {
	var target *store.Winch
	for i := range l.Winches {
		if l.Winches[i].ID == winchID {
			target = &l.Winches[i]
			break
		}
	}
	if target == nil {
		return ""
	}
	station := target.Station

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="240" viewBox="0 0 %g %g" font-family="system-ui,-apple-system,Segoe UI,Roboto,sans-serif">`, vbW, vbH)
	fmt.Fprintf(&b, `<rect x="0" y="0" width="%g" height="%g" fill="%s"/>`, vbW, vbH, colBg)
	fmt.Fprintf(&b, `<polygon points="%s" fill="%s" fill-opacity="0.6" stroke="%s" stroke-width="2"/>`,
		hullPoints(station), "#152238", colBorder)

	for i := range l.Winches {
		w := l.Winches[i]
		if w.Station != station {
			continue
		}
		writeWinch(&b, w, w.ID == winchID)
	}
	b.WriteString(`</svg>`)

	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(b.String()))
}

func writeWinch(b *strings.Builder, w store.Winch, highlight bool) {
	cx := w.X * vbW
	cy := w.Y * vbH
	bw, bh, inner := winchBoxDims(w.DrumCount)

	rad := float64(w.Orientation) * math.Pi / 180
	rotHalfH := math.Abs(math.Sin(rad))*(bw/2) + math.Abs(math.Cos(rad))*(bh/2)

	filledByIdx := make(map[int]bool, len(w.Drums))
	for _, d := range w.Drums {
		filledByIdx[d.Idx] = d.LineCount > 0
	}

	fmt.Fprintf(b, `<g transform="translate(%g %g) rotate(%d)">`, cx, cy, w.Orientation)

	// Winch body.
	bodyStroke, bodyWidth := colBorder, 1.5
	if highlight {
		bodyStroke, bodyWidth = colAccent, 3.0
	}
	fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" rx="8" fill="%s" stroke="%s" stroke-width="%g"/>`,
		-bw/2, -bh/2, bw, bh, colWinchFill, bodyStroke, bodyWidth)

	// Drum cells + numbers.
	for i := 0; i < w.DrumCount; i++ {
		filled := filledByIdx[i+1]
		cellX := -inner/2 + float64(i)*(drumW+drumGap)
		numCx := cellX + drumW/2
		cellFill := colBg
		numFill := colText
		if filled {
			cellFill = colAccentDeep
			numFill = "#fff"
		}
		fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" rx="3" fill="%s" stroke="%s" stroke-width="1"/>`,
			cellX, -drumH/2, drumW, drumH, cellFill, colBorder)
		fmt.Fprintf(b, `<text x="%g" y="0" fill="%s" font-size="13" font-weight="700" text-anchor="middle" dominant-baseline="central" transform="rotate(%d %g 0)">%d</text>`,
			numCx, numFill, -w.Orientation, numCx, i+1)
	}

	// Worst-case status mark (shape carries status, not just color).
	writeStatusMark(b, w.WorstStatus, bw/2-7, -bh/2+7, 9)

	// Drive-type marker (E electric / H hydraulic).
	drive := "E"
	if w.DriveType == "hydraulic" {
		drive = "H"
	}
	fmt.Fprintf(b, `<text x="%g" y="%g" fill="%s" font-size="13" font-weight="700" text-anchor="middle" transform="rotate(%d %g %g)">%s</text>`,
		-bw/2+9, -bh/2+15, colMuted, -w.Orientation, -bw/2+9, -bh/2+11, drive)

	// Label (counter-rotated to stay upright).
	fmt.Fprintf(b, `<text x="0" y="%g" fill="%s" font-size="14" font-weight="600" text-anchor="middle" transform="rotate(%d)">%s</text>`,
		rotHalfH+16, colText, -w.Orientation, xmlEscape(w.Label))

	b.WriteString(`</g>`)
}

func writeStatusMark(b *strings.Builder, status string, cx, cy, r float64) {
	fill := statusColor(status)
	switch condClass(status) {
	case "monitor":
		fmt.Fprintf(b, `<path d="M%g %gL%g %gL%g %gL%g %gZ" fill="%s" stroke="%s" stroke-width="1.25"/>`,
			cx, cy-r, cx+r, cy, cx, cy+r, cx-r, cy, fill, colBg)
	case "action":
		fmt.Fprintf(b, `<path d="M%g %gL%g %gL%g %gZ" fill="%s" stroke="%s" stroke-width="1.25"/>`,
			cx, cy-r, cx+r, cy+r, cx-r, cy+r, fill, colBg)
	default:
		fmt.Fprintf(b, `<circle cx="%g" cy="%g" r="%g" fill="%s" stroke="%s" stroke-width="1.25"/>`,
			cx, cy, r, fill, colBg)
	}
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}
