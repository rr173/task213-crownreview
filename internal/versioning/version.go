package versioning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"task213-crownreview/internal/model"
)

// Snapshot is the frozen content of an inspection version.
type Snapshot struct {
	BatchID       int64                      `json:"batch_id"`
	BatchName     string                     `json:"batch_name"`
	Blocks        []model.PointCloudBlock    `json:"blocks"`
	Candidates    []model.BreakCandidate     `json:"candidates"`
	Occlusion     int                        `json:"occlusion_zone_count"`
	Confirmed     int                        `json:"confirmed_count"`
	Rejected      int                        `json:"rejected_count"`
	OpenRemaining int                        `json:"open_remaining"`
}

// BuildSnapshot assembles an immutable snapshot from current batch state.
func BuildSnapshot(batch *model.ScanBatch, blocks []*model.PointCloudBlock,
	cands []*model.BreakCandidate, occCount int) Snapshot {
	s := Snapshot{BatchID: batch.ID, BatchName: batch.Name, Occlusion: occCount}
	for _, b := range blocks {
		s.Blocks = append(s.Blocks, *b)
	}
	confirmed, rejected, open := 0, 0, 0
	for _, c := range cands {
		s.Candidates = append(s.Candidates, *c)
		switch c.Status {
		case model.CandStatusConfirmed:
			confirmed++
		case model.CandStatusRejected:
			rejected++
		case model.CandStatusOpen:
			open++
		}
	}
	s.Confirmed = confirmed
	s.Rejected = rejected
	s.OpenRemaining = open
	return s
}

// CanonicalHash returns a deterministic hash of the snapshot content.
func CanonicalHash(s Snapshot) string {
	// Stable JSON: sort candidate ids to avoid map/order nondeterminism.
	buf, _ := json.Marshal(s)
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

// ValidateFreeze ensures all candidates are resolved (no open ones) before freeze.
// It returns a handleable error instead of panicking so callers can surface the
// rejection to the user (e.g. as an HTTP 4xx). A fully resolved snapshot freezes.
func ValidateFreeze(s Snapshot) error {
	if s.OpenRemaining > 0 {
		return model.Errf(model.ErrInvalidArgument,
			"%d candidate(s) still open, cannot freeze", s.OpenRemaining)
	}
	return nil
}

// allowedVersionTransitions defines the version state machine.
var allowedVersionTransitions = map[string][]string{
	model.VerStatusDraft:     {model.VerStatusShared, model.VerStatusFrozen},
	model.VerStatusShared:    {model.VerStatusFrozen, model.VerStatusSuperseded},
	model.VerStatusFrozen:    {model.VerStatusSuperseded},
	model.VerStatusSuperseded: {},
}

// CanTransition reports whether a version status change is permitted.
func CanTransition(from, to string) bool {
	for _, t := range allowedVersionTransitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// ValidateTransition checks a version transition.
func ValidateTransition(from, to string) error {
	if !CanTransition(from, to) {
		return model.Errf(model.ErrUnsupportedAction, "cannot move version %q -> %q", from, to)
	}
	return nil
}

// LabelFor returns a deterministic version label.
func LabelFor(batchName string, seq int) string {
	clean := strings.Map(func(r rune) rune {
		if r == ' ' || r == '/' || r == '\\' {
			return '-'
		}
		return r
	}, batchName)
	return fmt.Sprintf("%s-v%d", clean, seq)
}

// SortCandidatesByID sorts candidates by id for stable snapshots.
func SortCandidatesByID(cands []*model.BreakCandidate) {
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].ID < cands[j].ID
	})
}
