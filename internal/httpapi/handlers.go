package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
)

func (s *Server) handleCreateBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	if body.Name == "" {
		writeError(w, model.Errf(model.ErrInvalidArgument, "name required"))
		return
	}
	b, err := s.svc.CreateBatch(body.Name, body.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleListBatches(w http.ResponseWriter, r *http.Request) {
	bs, err := s.svc.ListBatches()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bs)
}

func (s *Server) handleGetBatch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	b, err := s.svc.GetBatch(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleUploadBlock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	var req UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	in := &pointcloud.BlockInput{TreeID: req.TreeID, CoordSys: req.CoordSys, Points: req.Points}
	b, err := s.svc.UploadBlock(id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleParseBatch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	blocks, err := s.svc.ListBlocks(id)
	if err != nil {
		writeError(w, err)
		return
	}
	results := []map[string]interface{}{}
	for _, b := range blocks {
		if b.Status != model.BlockStatusPending {
			continue
		}
		_, perr := s.svc.ParseBlock(b.ID)
		results = append(results, map[string]interface{}{
			"block_id": b.ID, "error": errToStr(perr),
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"parsed": results})
}

func (h *Server) handleParseBlock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	b, err := h.svc.ParseBlock(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handlePublishBatch(w http.ResponseWriter, r *http.Request) {
	_, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	b, err := s.svc.PublishBatch(0)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleBatchStats(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	blocks, layered, cands, confirmed, versions, err := s.svc.BatchStats(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"batch_id": id, "blocks": blocks, "layered": layered,
		"candidates": cands, "confirmed": confirmed, "versions": versions,
	})
}

func (s *Server) handleListBlocks(w http.ResponseWriter, r *http.Request) {
	batchID := int64(0)
	if v := r.URL.Query().Get("batch_id"); v != "" {
		var err error
		batchID, err = strconvParse(v)
		if err != nil {
			writeError(w, model.ErrInvalidArgument)
			return
		}
	}
	bs, err := s.svc.ListBlocks(batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bs)
}

func (s *Server) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	b, err := s.svc.GetBlock(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleBlockSkeleton(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	edges, err := s.svc.GetSkeleton(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, edges)
}

func (s *Server) handleBlockOcclusion(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	zones, err := s.svc.GetOcclusion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, zones)
}

func (s *Server) handleListCandidates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	blockID := int64(0)
	if v := q.Get("block_id"); v != "" {
		var err error
		blockID, err = strconvParse(v)
		if err != nil {
			writeError(w, model.ErrInvalidArgument)
			return
		}
	}
	status := q.Get("status")
	cs, err := s.svc.ListCandidates(blockID, status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *Server) handleGetCandidate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	c, err := s.svc.GetCandidate(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleConfirmCandidate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	c, err := s.svc.ConfirmCandidate(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleRejectCandidate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	c, err := s.svc.RejectCandidate(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleMergeCandidate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	var req MergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	res, err := s.svc.MergeCandidate(id, req.TargetID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleAddOpinion(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	var req OpinionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	o, err := s.svc.AddOpinion(id, req.VersionID, req.Reviewer, req.Opinion, req.Detail)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) handleListOpinions(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	os, err := s.svc.ListOpinions(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, os)
}

func (s *Server) handleCreateVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	v, err := s.svc.CreateVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	vs, err := s.svc.ListVersions(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vs)
}

func (s *Server) handleFreezeVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	v, err := s.svc.FreezeVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleShareVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	v, err := s.svc.ShareVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleSupersedeVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, model.ErrInvalidArgument)
		return
	}
	v, err := s.svc.SupersedeVersion(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok", Module: "task213-crownreview"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.svc.Stats()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func errToStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// strconvParse parses a base-10 int64.
func strconvParse(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
