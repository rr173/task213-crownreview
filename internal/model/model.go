package model

import (
	"errors"
	"fmt"
	"time"
)

// Domain errors.
var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrBadFormat         = errors.New("bad point format")
	ErrUnknownCoordSys   = errors.New("unknown coordinate system")
	ErrTreeIDConflict    = errors.New("tree id conflict")
	ErrFrozenWrite       = errors.New("frozen version write rejected")
	ErrDuplicateBlock    = errors.New("duplicate block hash")
	ErrParseInProgress   = errors.New("block parse already in progress")
	ErrUnsupportedAction = errors.New("unsupported action for current status")
)

// Status constants for scan batch.
const (
	BatchStatusUploading  = "uploading"  // 上传中
	BatchStatusProcessing = "processing" // 处理中
	BatchStatusReviewing  = "reviewing"  // 待复核
	BatchStatusPublished  = "published"  // 已发布
)

// Status constants for point cloud block.
const (
	BlockStatusPending   = "pending"   // 待解析
	BlockStatusLayered   = "layered"   // 已分层
	BlockStatusMissing   = "missing"   // 缺失
	BlockStatusDuplicate = "duplicate" // 重复
)

// Status constants for break candidate.
const (
	CandStatusGenerated = "generated" // 生成
	CandStatusOpen      = "open"      // 待确认
	CandStatusMerged    = "merged"    // 合并
	CandStatusRejected  = "rejected"  // 否决
	CandStatusConfirmed = "confirmed" // 确认
)

// Status constants for inspection version.
const (
	VerStatusDraft     = "draft"     // 草稿
	VerStatusShared    = "shared"    // 共享
	VerStatusFrozen    = "frozen"    // 冻结
	VerStatusSuperseded = "superseded" // 替代
)

// ScanBatch is the top-level container grouping point cloud blocks of one survey.
type ScanBatch struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// PointCloudBlock is one tree-split point cloud chunk.
type PointCloudBlock struct {
	ID           int64     `json:"id"`
	BatchID      int64     `json:"batch_id"`
	TreeID       string    `json:"tree_id"`
	BlockHash    string    `json:"block_hash"`
	CoordSys     string    `json:"coord_sys"`
	Status       string    `json:"status"`
	PointCount   int       `json:"point_count"`
	BBoxMin      [3]float64 `json:"bbox_min"`
	BBoxMax      [3]float64 `json:"bbox_max"`
	Summary      string    `json:"summary"`
	CreatedAt    time.Time `json:"created_at"`
	ParsedAt     *time.Time `json:"parsed_at,omitempty"`
}

// SkeletonEdge is one connectivity edge of the branch skeleton graph.
type SkeletonEdge struct {
	ID        int64      `json:"id"`
	BlockID   int64      `json:"block_id"`
	FromNode  string     `json:"from_node"`
	ToNode    string     `json:"to_node"`
	FromPoint [3]float64 `json:"from_point"`
	ToPoint   [3]float64 `json:"to_point"`
	Radius    float64    `json:"radius"`
	CreatedAt time.Time  `json:"created_at"`
}

// XYZ returns the point as a 3-element array.
func (p Point3) XYZ() [3]float64 { return [3]float64{p.X, p.Y, p.Z} }

// BreakCandidate is a suspected branch break with supporting evidence.
type BreakCandidate struct {
	ID          int64     `json:"id"`
	BlockID     int64     `json:"block_id"`
	TreeID      string    `json:"tree_id"`
	Status      string    `json:"status"`
	Position    [3]float64 `json:"position"`
	Severity    float64   `json:"severity"`
	Confidence  float64   `json:"confidence"`
	EdgeA       string    `json:"edge_a"`
	EdgeB       string    `json:"edge_b"`
	Reason      string    `json:"reason"`
	MergedInto  *int64    `json:"merged_into,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// OcclusionZone is an uncertain region caused by sensor occlusion.
type OcclusionZone struct {
	ID        int64     `json:"id"`
	BlockID   int64     `json:"block_id"`
	Center    [3]float64 `json:"center"`
	Radius    float64   `json:"radius"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

// ReviewOpinion is an expert's opinion on a candidate, tied to a version.
type ReviewOpinion struct {
	ID           int64     `json:"id"`
	CandidateID  int64     `json:"candidate_id"`
	VersionID    int64     `json:"version_id"`
	Reviewer     string    `json:"reviewer"`
	Opinion      string    `json:"opinion"` // confirm|reject|merge|note
	Detail       string    `json:"detail"`
	CreatedAt    time.Time `json:"created_at"`
}

// InspectionVersion is an immutable published snapshot of a batch review.
type InspectionVersion struct {
	ID           int64     `json:"id"`
	BatchID      int64     `json:"batch_id"`
	Status       string    `json:"status"`
	Label        string    `json:"label"`
	Snapshot     string    `json:"snapshot"` // frozen JSON summary
	CanonicalHash string   `json:"canonical_hash"`
	CreatedAt    time.Time `json:"created_at"`
	FrozenAt     *time.Time `json:"frozen_at,omitempty"`
}

// Point3 is a single 3D point with optional intensity.
type Point3 struct {
	X, Y, Z  float64
	Intensity float64
}

// Errf builds a wrapped domain error with context.
func Errf(base error, format string, args ...any) error {
	return fmt.Errorf("%w: %s", base, fmt.Sprintf(format, args...))
}
