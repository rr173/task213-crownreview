package review

import (
	"task213-crownreview/internal/model"
)

// ValidOpinions enumerates accepted opinion kinds.
var ValidOpinions = map[string]bool{
	"confirm": true,
	"reject":  true,
	"merge":   true,
	"note":    true,
}

// ValidateOpinion checks an incoming opinion payload.
func ValidateOpinion(opinion, reviewer string) error {
	if !ValidOpinions[opinion] {
		return model.Errf(model.ErrInvalidArgument, "unknown opinion %q", opinion)
	}
	if reviewer == "" {
		return model.Errf(model.ErrInvalidArgument, "reviewer required")
	}
	return nil
}

// OpinionSummary aggregates opinions for a version.
type OpinionSummary struct {
	VersionID   int64
	Confirmed   int
	Rejected    int
	Merged      int
	Notes       int
	ByCandidate map[int64]int
}

// Summarize counts opinions across a candidate set.
func Summarize(items []*model.ReviewOpinion) OpinionSummary {
	s := OpinionSummary{ByCandidate: map[int64]int{}}
	for _, o := range items {
		s.ByCandidate[o.CandidateID]++
		switch o.Opinion {
		case "confirm":
			s.Confirmed++
		case "reject":
			s.Rejected++
		case "merge":
			s.Merged++
		case "note":
			s.Notes++
		}
	}
	return s
}
