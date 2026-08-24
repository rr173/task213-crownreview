package versioning

import (
	"testing"

	"task213-crownreview/internal/model"
)

func TestCanonicalHashStable(t *testing.T) {
	s1 := Snapshot{BatchID: 1, BatchName: "a", Occlusion: 2, Confirmed: 3}
	s2 := Snapshot{BatchID: 1, BatchName: "a", Occlusion: 2, Confirmed: 3}
	if CanonicalHash(s1) != CanonicalHash(s2) {
		t.Fatalf("identical snapshots produced different hashes")
	}
	s2.Confirmed = 4
	if CanonicalHash(s1) == CanonicalHash(s2) {
		t.Fatalf("different snapshots produced identical hashes")
	}
}

func TestValidateFreezeRejectsOpen(t *testing.T) {
	s := Snapshot{OpenRemaining: 1}
	if err := ValidateFreeze(s); err == nil {
		t.Fatalf("expected freeze to reject snapshot with open candidates")
	}
	s.OpenRemaining = 0
	if err := ValidateFreeze(s); err != nil {
		t.Fatalf("freeze should accept fully resolved snapshot: %v", err)
	}
}

func TestVersionTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{model.VerStatusDraft, model.VerStatusFrozen, true},
		{model.VerStatusDraft, model.VerStatusShared, true},
		{model.VerStatusFrozen, model.VerStatusSuperseded, true},
		{model.VerStatusSuperseded, model.VerStatusFrozen, false},
		{model.VerStatusFrozen, model.VerStatusDraft, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.ok {
			t.Errorf("CanTransition(%s,%s)=%v want %v", c.from, c.to, got, c.ok)
		}
	}
}
