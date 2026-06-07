package domain

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestWorst(t *testing.T) {
	cases := []struct {
		in   []Condition
		want Condition
	}{
		{[]Condition{Good, Good}, Good},
		{[]Condition{Good, Monitor}, Monitor},
		{[]Condition{Monitor, Action, Good}, Action},
		{[]Condition{"", Good}, Good},
		{[]Condition{}, ""},
		{[]Condition{"", ""}, ""},
	}
	for _, c := range cases {
		if got := Worst(c.in...); got != c.want {
			t.Errorf("Worst(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLiveSideAge(t *testing.T) {
	now := date(2026, 6, 1)
	change := date(2026, 5, 2) // 30 days before now

	// Active side: accumulated + elapsed since change.
	if got := LiveSideAge(100, &change, true, now); got != 130 {
		t.Errorf("active live age = %d, want 130", got)
	}
	// Inactive side: frozen accumulator only.
	if got := LiveSideAge(100, &change, false, now); got != 100 {
		t.Errorf("inactive live age = %d, want 100", got)
	}
	// No change date: just the accumulator.
	if got := LiveSideAge(40, nil, true, now); got != 40 {
		t.Errorf("nil change live age = %d, want 40", got)
	}
}

func TestTurnDue(t *testing.T) {
	if TurnDue(false, 1000) {
		t.Error("non-turnable line should never be turn-due")
	}
	if TurnDue(true, TurnIntervalDays-1) {
		t.Error("under interval should not be due")
	}
	if !TurnDue(true, TurnIntervalDays) {
		t.Error("at interval should be due")
	}
}

func TestNextInspectionDue(t *testing.T) {
	if NextInspectionDue(nil) != nil {
		t.Error("nil installation -> nil due")
	}
	inst := date(2026, 1, 1)
	got := NextInspectionDue(&inst)
	if got == nil || !got.Equal(date(2026, 1, 31)) {
		t.Errorf("due = %v, want 2026-01-31", got)
	}
}
