package domain

import "time"

// Schedule constants from the manufacturer cadence (spec §1).
const (
	TurnIntervalDays           = 182 // ~6 months
	PositionChangeIntervalDays = 365 // 12 months
	InspectionIntervalDays     = 30  // monthly PMS inspection
)

// DaysBetween returns whole days from a to b (b - a), never negative.
func DaysBetween(a, b time.Time) int {
	d := int(b.Sub(a).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// AgeDays returns days from a date until now. Returns 0 for the zero date.
func AgeDays(date *time.Time, now time.Time) int {
	if date == nil || date.IsZero() {
		return 0
	}
	return DaysBetween(*date, now)
}

// LiveSideAge is the accumulated age for a side: frozen accumulator plus, if this is
// the active side, the days elapsed since it last became active. Age accrues only on
// the side in use (spec §4.4).
func LiveSideAge(accumulatedDays int, changeDate *time.Time, isActive bool, now time.Time) int {
	age := accumulatedDays
	if isActive && changeDate != nil && !changeDate.IsZero() {
		age += DaysBetween(*changeDate, now)
	}
	return age
}

// NextInspectionDue is a placeholder until the inspections slice supplies the real
// last-inspection date: monthly cadence from installation. Returns nil if unknown.
func NextInspectionDue(installation *time.Time) *time.Time {
	if installation == nil || installation.IsZero() {
		return nil
	}
	due := installation.AddDate(0, 0, InspectionIntervalDays)
	return &due
}

// TurnDue reports whether a reversible line is due to be turned: the active side has
// accrued at least the turn interval.
func TurnDue(canBeTurned bool, activeSideAgeDays int) bool {
	return canBeTurned && activeSideAgeDays >= TurnIntervalDays
}
