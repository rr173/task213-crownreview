package review

import (
	"testing"

	"task213-crownreview/internal/model"
)

func TestCandidateTransitionMatrix(t *testing.T) {
	cases := []struct {
		from string
		to   string
		want bool
	}{
		{model.CandStatusOpen, model.CandStatusConfirmed, true},
		{model.CandStatusOpen, model.CandStatusRejected, true},
		{model.CandStatusOpen, model.CandStatusMerged, true},
		{model.CandStatusConfirmed, model.CandStatusRejected, false},
		{model.CandStatusRejected, model.CandStatusConfirmed, false},
	}
	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestValidateOpinionRequiresKnownKindAndReviewer(t *testing.T) {
	if err := ValidateOpinion("note", "analyst"); err != nil {
		t.Fatalf("valid opinion rejected: %v", err)
	}
	if err := ValidateOpinion("unknown", "analyst"); err == nil {
		t.Fatal("unknown opinion accepted")
	}
	if err := ValidateOpinion("note", ""); err == nil {
		t.Fatal("opinion without reviewer accepted")
	}
}
