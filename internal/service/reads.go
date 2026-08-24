package service

import (
	"task213-crownreview/internal/model"
)

// CreateBatch creates a new scan batch.
func (s *Service) CreateBatch(name, description string) (*model.ScanBatch, error) {
	return s.store.CreateBatch(name, description)
}

// GetBatch fetches a batch by id.
func (s *Service) GetBatch(id int64) (*model.ScanBatch, error) {
	return s.store.GetBatch(id)
}

// ListBatches returns all batches.
func (s *Service) ListBatches() ([]*model.ScanBatch, error) {
	return s.store.ListBatches()
}

// GetBlock fetches a block by id.
func (s *Service) GetBlock(id int64) (*model.PointCloudBlock, error) {
	return s.store.GetBlock(id)
}

// ListBlocks returns blocks (all or for a batch).
func (s *Service) ListBlocks(batchID int64) ([]*model.PointCloudBlock, error) {
	return s.store.ListBlocks(batchID)
}

// GetSkeleton returns skeleton edges of a block.
func (s *Service) GetSkeleton(blockID int64) ([]model.SkeletonEdge, error) {
	return s.store.GetSkeleton(blockID)
}

// GetOcclusion returns occlusion zones of a block.
func (s *Service) GetOcclusion(blockID int64) ([]model.OcclusionZone, error) {
	return s.store.GetOcclusion(blockID)
}

// GetCandidate fetches a candidate by id.
func (s *Service) GetCandidate(id int64) (*model.BreakCandidate, error) {
	return s.store.GetCandidate(id)
}

// ListCandidates returns candidates filtered by block and/or status.
func (s *Service) ListCandidates(blockID int64, status string) ([]*model.BreakCandidate, error) {
	return s.store.ListCandidates(blockID, status)
}

// ListOpinions returns opinions of a candidate.
func (s *Service) ListOpinions(candidateID int64) ([]*model.ReviewOpinion, error) {
	return s.store.ListOpinions(candidateID)
}

// ListVersions returns versions of a batch.
func (s *Service) ListVersions(batchID int64) ([]*model.InspectionVersion, error) {
	return s.store.ListVersions(batchID)
}

// FrozenVersionForBatch returns the latest frozen version of a batch.
func (s *Service) FrozenVersionForBatch(batchID int64) (*model.InspectionVersion, error) {
	return s.store.FrozenVersionForBatch(batchID)
}

// BatchStats aggregates batch statistics.
func (s *Service) BatchStats(batchID int64) (blocks, layered, candidates, confirmed, versions int, err error) {
	return s.store.BatchStats(batchID)
}
