package services

import (
	"testing"

	"github.com/google/uuid"
)

func TestCanDecide(t *testing.T) {
	creator, manager, other := uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		name     string
		actor    uuid.UUID
		managers []uuid.UUID
		full     bool
		super    bool
		want     bool
	}{
		{"own transaction is not allowed, even for owner/admin", creator, nil, true, false, false},
		{"own transaction is allowed for a Super Administrator", creator, []uuid.UUID{manager}, true, true, true},
		{"no managers: anyone with the permission", other, nil, false, false, true},
		{"assigned manager", manager, []uuid.UUID{manager}, false, false, true},
		{"not a manager", other, []uuid.UUID{manager}, false, false, false},
		{"owner/admin overrides", other, []uuid.UUID{manager}, true, false, true},
	}
	for _, c := range cases {
		if got := canDecide(creator, c.actor, c.managers, c.full, c.super); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
