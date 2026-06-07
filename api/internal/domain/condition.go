// Package domain holds business rules and computed values that must never be
// hand-edited or duplicated in the client (spec §2, §7).
package domain

// Condition is the inspection condition scale (confirm exact scale with NCL).
type Condition string

const (
	Good    Condition = "Good"
	Monitor Condition = "Monitor"
	Action  Condition = "Action"
)

// severity orders conditions worst-last so Worst can pick the most severe.
func severity(c Condition) int {
	switch c {
	case Action:
		return 3
	case Monitor:
		return 2
	case Good:
		return 1
	default:
		return 0 // unknown / no condition
	}
}

// Worst returns the most severe condition among the inputs (the worst-case status
// dot shown on a winch/storage symbol). Empty/unknown values are ignored; if nothing
// usable is present it returns "".
func Worst(conditions ...Condition) Condition {
	worst := Condition("")
	best := 0
	for _, c := range conditions {
		if s := severity(c); s > best {
			best = s
			worst = c
		}
	}
	return worst
}
