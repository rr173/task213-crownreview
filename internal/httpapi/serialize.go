package httpapi

import "task213-crownreview/internal/model"

// ModuleName identifies this service module in self-check responses.
const ModuleName = "crownreview"

// APIError is the JSON error envelope.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UploadRequest is the body for block upload.
type UploadRequest struct {
	TreeID   string        `json:"tree_id"`
	CoordSys string        `json:"coord_sys"`
	Points   []model.Point3 `json:"points"`
}

// OpinionRequest is the body for adding a review opinion.
type OpinionRequest struct {
	VersionID int64  `json:"version_id"`
	Reviewer  string `json:"reviewer"`
	Opinion   string `json:"opinion"`
	Detail    string `json:"detail"`
}

// MergeRequest is the body for merging a candidate.
type MergeRequest struct {
	TargetID int64 `json:"target_id"`
}

// HealthResponse reports service liveness.
type HealthResponse struct {
	Status string `json:"status"`
	Module string `json:"module"`
}
