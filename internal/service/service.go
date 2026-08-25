package service

import (
	"encoding/json"

	"task213-crownreview/internal/detection"
	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/review"
	"task213-crownreview/internal/skeleton"
	"task213-crownreview/internal/store"
	"task213-crownreview/internal/versioning"
)

// Service orchestrates the domain workflow over the store.
type Service struct {
	store *store.DB
}

// New creates a service backed by the given store.
func New(db *store.DB) *Service { return &Service{store: db} }

// UploadBlock validates and stores a point cloud block. Idempotent by hash.
func (s *Service) UploadBlock(batchID int64, in *pointcloud.BlockInput) (*model.PointCloudBlock, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	batch, err := s.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	hash := pointcloud.Hash(in.TreeID, in.CoordSys, in.Points)
	// Idempotency wins: same block hash returns the existing block (no new row),
	// regardless of the batch's current status (a re-upload is a no-op).
	if existing, berr := s.store.BlockByHash(hash); berr == nil {
		return existing, nil
	}
	if batch.Status != model.BatchStatusUploading && batch.Status != model.BatchStatusProcessing {
		return nil, model.Errf(model.ErrInvalidArgument, "batch not in uploadable state: %s", batch.Status)
	}
	// tree id uniqueness within batch (different hash for an already-owned tree).
	if existing, ok, _ := s.store.BlockExistsByTree(batchID, in.TreeID); ok {
		return nil, model.Errf(model.ErrTreeIDConflict,
			"tree %q already owned by block %d", in.TreeID, existing.ID)
	}
	min, max := pointcloud.BoundingBox(in.Points)
	blk := &model.PointCloudBlock{
		BatchID:    batchID,
		TreeID:     in.TreeID,
		BlockHash:  hash,
		CoordSys:   in.CoordSys,
		PointCount: len(in.Points),
		BBoxMin:    min,
		BBoxMax:    max,
		Summary:    pointcloud.Summary(in.TreeID, in.Points),
	}
	// move batch to processing on first real block
	_ = s.store.SetBatchStatus(batchID, model.BatchStatusProcessing, false)
	return s.store.CreateBlock(blk, in.Points, false)
}

// ParseBlock builds the skeleton, detects breaks and occlusion, and stores results.
// Block parsing may run in parallel for different blocks; single-tree skeleton
// merge is inherently serial per block.
func (s *Service) ParseBlock(blockID int64) (*model.PointCloudBlock, error) {
	blk, err := s.store.GetBlock(blockID)
	if err != nil {
		return nil, err
	}
	if blk.Status == model.BlockStatusMissing || blk.Status == model.BlockStatusDuplicate {
		return nil, model.Errf(model.ErrInvalidArgument, "block %d not parseable (%s)", blockID, blk.Status)
	}
	if blk.PointCount == 0 {
		return nil, model.Errf(model.ErrInvalidArgument, "block %d has no points", blockID)
	}
	pts, err := s.store.GetBlockPoints(blockID)
	if err != nil {
		return nil, err
	}
	if len(pts) == 0 {
		// nothing to parse => mark missing
		_ = s.store.SetBlockStatus(blockID, model.BlockStatusMissing)
		return s.store.GetBlock(blockID)
	}
	g := skeleton.Build(pts, skeleton.DefaultOptions())
	edges := g.Edges()
	if err := s.store.SaveSkeleton(blockID, edges); err != nil {
		return nil, err
	}
	cands := detection.DetectBreaks(g, detection.DefaultDetectOptions())
	for i := range cands {
		cands[i].BlockID = blockID
		cands[i].TreeID = blk.TreeID
		if _, err := s.store.CreateCandidate(&cands[i]); err != nil {
			return nil, err
		}
	}
	zones := detection.DetectOcclusion(pts, 0.05, 3, 2)
	if err := s.store.SaveOcclusion(blockID, zones); err != nil {
		return nil, err
	}
	if err := s.store.SetBlockStatus(blockID, model.BlockStatusLayered); err != nil {
		return nil, err
	}
	// batch moves to reviewing once any block is layered
	_ = s.store.SetBatchStatus(blk.BatchID, model.BatchStatusReviewing, false)
	return s.store.GetBlock(blockID)
}

// ConfirmCandidate marks a candidate as confirmed.
func (s *Service) ConfirmCandidate(id int64) (*model.BreakCandidate, error) {
	c, err := s.store.GetCandidate(id)
	if err != nil {
		return nil, err
	}
	if err := review.ValidateConfirm(c.Status); err != nil {
		return nil, err
	}
	if err := s.store.SetCandidateStatus(id, model.CandStatusConfirmed, nil); err != nil {
		return nil, err
	}
	return s.store.GetCandidate(id)
}

// RejectCandidate marks a candidate as rejected.
func (s *Service) RejectCandidate(id int64) (*model.BreakCandidate, error) {
	c, err := s.store.GetCandidate(id)
	if err != nil {
		return nil, err
	}
	if err := review.ValidateReject(c.Status); err != nil {
		return nil, err
	}
	if err := s.store.SetCandidateStatus(id, model.CandStatusRejected, nil); err != nil {
		return nil, err
	}
	return s.store.GetCandidate(id)
}

// MergeCandidate merges src into an already-confirmed dst.
func (s *Service) MergeCandidate(srcID, dstID int64) (review.MergeResult, error) {
	src, err := s.store.GetCandidate(srcID)
	if err != nil {
		return review.MergeResult{}, err
	}
	dst, err := s.store.GetCandidate(dstID)
	if err != nil {
		return review.MergeResult{}, err
	}
	if err := review.ValidateMerge(src.Status, dst.Status); err != nil {
		return review.MergeResult{}, err
	}
	if err := s.store.SetCandidateStatus(srcID, model.CandStatusMerged, &dstID); err != nil {
		return review.MergeResult{}, err
	}
	return review.ApplyMerge(srcID, dstID, src.TreeID, dst.TreeID), nil
}

// AddOpinion records a review opinion on a candidate for a version.
func (s *Service) AddOpinion(candidateID, versionID int64, reviewer, opinion, detail string) (*model.ReviewOpinion, error) {
	if err := review.ValidateOpinion(opinion, reviewer); err != nil {
		return nil, err
	}
	if _, err := s.store.GetCandidate(candidateID); err != nil {
		return nil, err
	}
	if _, err := s.store.GetVersion(versionID); err != nil {
		return nil, err
	}
	o := &model.ReviewOpinion{
		CandidateID: candidateID, VersionID: versionID,
		Reviewer: reviewer, Opinion: opinion, Detail: detail,
	}
	return s.store.AddOpinion(o)
}

// CreateVersion builds and stores a draft inspection version snapshot.
func (s *Service) CreateVersion(batchID int64) (*model.InspectionVersion, error) {
	batch, err := s.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	blocks, err := s.store.ListBlocks(batchID)
	if err != nil {
		return nil, err
	}
	cands, err := s.store.CandidatesForBatch(batchID)
	if err != nil {
		return nil, err
	}
	occ := 0
	for _, b := range blocks {
		zs, _ := s.store.GetOcclusion(b.ID)
		occ += len(zs)
	}
	versioning.SortCandidatesByID(cands)
	snap := versioning.BuildSnapshot(batch, blocks, cands, occ)
	hash := versioning.CanonicalHash(snap)
	label := versioning.LabelFor(batch.Name, len(blocks))
	buf, _ := json.Marshal(snap)
	return s.store.CreateVersion(batchID, label, string(buf), hash)
}

// FreezeVersion validates and freezes a version.
func (s *Service) FreezeVersion(versionID int64) (*model.InspectionVersion, error) {
	v, err := s.store.GetVersion(versionID)
	if err != nil {
		return nil, err
	}
	if err := versioning.ValidateTransition(v.Status, model.VerStatusFrozen); err != nil {
		return nil, err
	}
	cands, err := s.store.CandidatesForBatch(v.BatchID)
	if err != nil {
		return nil, err
	}
	versioning.SortCandidatesByID(cands)
	batch, _ := s.store.GetBatch(v.BatchID)
	blocks, _ := s.store.ListBlocks(v.BatchID)
	occ := 0
	for _, b := range blocks {
		zs, _ := s.store.GetOcclusion(b.ID)
		occ += len(zs)
	}
	snap := versioning.BuildSnapshot(batch, blocks, cands, occ)
	if err := versioning.ValidateFreeze(snap); err != nil {
		return nil, err
	}
	// recompute canonical hash at freeze time
	hash := versioning.CanonicalHash(snap)
	if err := s.store.SetVersionCanonical(versionID, hash); err != nil {
		return nil, err
	}
	if err := s.store.SetVersionStatus(versionID, model.VerStatusFrozen); err != nil {
		return nil, err
	}
	return s.store.GetVersion(versionID)
}

// ShareVersion moves a version to shared.
func (s *Service) ShareVersion(versionID int64) (*model.InspectionVersion, error) {
	v, err := s.store.GetVersion(versionID)
	if err != nil {
		return nil, err
	}
	if err := versioning.ValidateTransition(v.Status, model.VerStatusShared); err != nil {
		return nil, err
	}
	if err := s.store.SetVersionStatus(versionID, model.VerStatusShared); err != nil {
		return nil, err
	}
	return s.store.GetVersion(versionID)
}

// SupersedeVersion freezes (if needed) and supersedes a version.
func (s *Service) SupersedeVersion(versionID int64) (*model.InspectionVersion, error) {
	v, err := s.store.GetVersion(versionID)
	if err != nil {
		return nil, err
	}
	if err := versioning.ValidateTransition(v.Status, model.VerStatusSuperseded); err != nil {
		return nil, err
	}
	if err := s.store.SetVersionStatus(versionID, model.VerStatusSuperseded); err != nil {
		return nil, err
	}
	return s.store.GetVersion(versionID)
}

// PublishBatch publishes a batch only if a frozen version exists.
func (s *Service) PublishBatch(batchID int64) (*model.ScanBatch, error) {
	batch, err := s.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if batch.Status == model.BatchStatusPublished {
		return batch, nil
	}
	frozen, err := s.store.FrozenVersionForBatch(batchID)
	if err != nil {
		return nil, err
	}
	if frozen == nil {
		return nil, model.Errf(model.ErrInvalidArgument, "no frozen version, cannot publish")
	}
	if frozen.BatchID != batchID {
		// Defensive guard: a frozen version for a different batch must never
		// satisfy this batch's publish request.
		return nil, model.Errf(model.ErrInvalidArgument, "frozen version belongs to a different batch, cannot publish")
	}
	if err := s.store.SetBatchStatus(batchID, model.BatchStatusPublished, true); err != nil {
		return nil, err
	}
	return s.store.GetBatch(batchID)
}

// Stats returns global service statistics.
func (s *Service) Stats() (map[string]int, error) {
	out := map[string]int{}
	batches, _ := s.store.ListBatches()
	out["batches"] = len(batches)
	blocks, _ := s.store.ListBlocks(0)
	out["blocks"] = len(blocks)
	cands, _ := s.store.ListCandidates(0, "")
	out["candidates"] = len(cands)
	confirmed, _ := s.store.ListCandidates(0, model.CandStatusConfirmed)
	out["confirmed"] = len(confirmed)
	rejected, _ := s.store.ListCandidates(0, model.CandStatusRejected)
	out["rejected"] = len(rejected)
	return out, nil
}
