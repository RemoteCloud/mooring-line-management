package store

import (
	"errors"
	"testing"
)

// validateMoveRequest must accept exactly one destination and reject both-empty
// and both-set (the two ways MoveLine previously misbehaved: silent clear / 500).
func TestValidateMoveRequest(t *testing.T) {
	cases := []struct {
		name              string
		drumID, storageID string
		wantErr           bool
	}{
		{"drum only", "drum-1", "", false},
		{"storage only", "", "store-1", false},
		{"both empty", "", "", true},
		{"both set", "drum-1", "store-1", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateMoveRequest(c.drumID, c.storageID)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidMoveTarget) {
					t.Fatalf("want ErrInvalidMoveTarget, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}
