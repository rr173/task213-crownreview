package review

import (
	"task213-crownreview/internal/model"
)

// allowedTransitions defines the candidate state machine.
//
// Once a candidate is confirmed it represents an expert's final verdict and
// is treated as resolved: it can only be merged into another confirmed
// candidate. In particular it may no longer be rejected.
var allowedTransitions = map[string][]string{
	model.CandStatusOpen:      {model.CandStatusConfirmed, model.CandStatusRejected, model.CandStatusMerged},
	model.CandStatusGenerated: {model.CandStatusOpen, model.CandStatusRejected},
	model.CandStatusMerged:    {},
	model.CandStatusRejected:  {},
	model.CandStatusConfirmed: {},
}

// CanTransition reports whether a status change is permitted.
func CanTransition(from, to string) bool {
	for _, t := range allowedTransitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// ValidateConfirm checks a confirm action.
func ValidateConfirm(current string) error {
	if !CanTransition(current, model.CandStatusConfirmed) {
		return model.Errf(model.ErrUnsupportedAction, "cannot confirm from %q", current)
	}
	return nil
}

// ValidateReject checks a reject action.
func ValidateReject(current string) error {
	if !CanTransition(current, model.CandStatusRejected) {
		return model.Errf(model.ErrUnsupportedAction, "cannot reject from %q", current)
	}
	return nil
}

// ValidateMerge checks a merge action and resolves target.
func ValidateMerge(current, targetStatus string) error {
	if !CanTransition(current, model.CandStatusMerged) {
		return model.Errf(model.ErrUnsupportedAction, "cannot merge from %q", current)
	}
	if targetStatus != model.CandStatusConfirmed {
		return model.Errf(model.ErrInvalidArgument, "merge target must be confirmed, got %q", targetStatus)
	}
	return nil
}

// MergeResult describes the outcome of merging src into dst.
type MergeResult struct {
	SourceID   int64
	TargetID   int64
	SourceTree string
	TargetTree string
}

// ApplyMerge marks source as merged into target.
func ApplyMerge(srcID, dstID int64, srcTree, dstTree string) MergeResult {
	return MergeResult{SourceID: srcID, TargetID: dstID, SourceTree: srcTree, TargetTree: dstTree}
}
