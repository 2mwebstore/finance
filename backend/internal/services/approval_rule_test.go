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
		want     bool
	}{
		{"own transaction is never allowed", creator, nil, true, false},
		{"no managers: anyone with the permission", other, nil, false, true},
		{"assigned manager", manager, []uuid.UUID{manager}, false, true},
		{"not a manager", other, []uuid.UUID{manager}, false, false},
		{"owner/admin overrides", other, []uuid.UUID{manager}, true, true},
	}
	for _, c := range cases {
		if got := canDecide(creator, c.actor, c.managers, c.full); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
