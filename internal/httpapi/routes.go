package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
	"task213-crownreview/internal/webui"
)

// Server is the HTTP API server.
type Server struct {
	svc  *service.Service
	db   *store.DB
	addr string
	dbPath string
}

// NewServer builds the API server.
func NewServer(svc *service.Service, db *store.DB, addr, dbPath string) *Server {
	return &Server{svc: svc, db: db, addr: addr, dbPath: dbPath}
}

// Handler returns the configured http.Handler (routes under /api).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// batch
	mux.HandleFunc("POST /api/batches", s.handleCreateBatch)
	mux.HandleFunc("GET /api/batches", s.handleListBatches)
	mux.HandleFunc("GET /api/batches/{id}", s.handleGetBatch)
	mux.HandleFunc("POST /api/batches/{id}/upload", s.handleUploadBlock)
	mux.HandleFunc("POST /api/batches/{id}/parse", s.handleParseBatch)
	mux.HandleFunc("POST /api/batches/{id}/publish", s.handlePublishBatch)
	mux.HandleFunc("GET /api/batches/{id}/stats", s.handleBatchStats)

	// block
	mux.HandleFunc("GET /api/blocks", s.handleListBlocks)
	mux.HandleFunc("GET /api/blocks/{id}", s.handleGetBlock)
	mux.HandleFunc("POST /api/blocks/{id}/parse", s.handleParseBlock)
	mux.HandleFunc("GET /api/blocks/{id}/skeleton", s.handleBlockSkeleton)
	mux.HandleFunc("GET /api/blocks/{id}/occlusion", s.handleBlockOcclusion)

	// candidate
	mux.HandleFunc("GET /api/candidates", s.handleListCandidates)
	mux.HandleFunc("GET /api/candidates/{id}", s.handleGetCandidate)
	mux.HandleFunc("POST /api/candidates/{id}/confirm", s.handleConfirmCandidate)
	mux.HandleFunc("POST /api/candidates/{id}/reject", s.handleRejectCandidate)
	mux.HandleFunc("POST /api/candidates/{id}/merge", s.handleMergeCandidate)
	mux.HandleFunc("POST /api/candidates/{id}/opinions", s.handleAddOpinion)
	mux.HandleFunc("GET /api/candidates/{id}/opinions", s.handleListOpinions)

	// version
	mux.HandleFunc("POST /api/batches/{id}/versions", s.handleCreateVersion)
	mux.HandleFunc("GET /api/batches/{id}/versions", s.handleListVersions)
	mux.HandleFunc("POST /api/versions/{id}/freeze", s.handleFreezeVersion)
	mux.HandleFunc("POST /api/versions/{id}/share", s.handleShareVersion)
	mux.HandleFunc("POST /api/versions/{id}/supersede", s.handleSupersedeVersion)

	// selfcheck
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/stats", s.handleStats)

	// web view (embedded single-page 3D review UI)
	mux.Handle("GET /", webui.Handler())

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	code := "error"
	status := http.StatusBadRequest
	msg := err.Error()
	// map domain errors to HTTP status
	switch {
	case err == nil:
		return
	}
	if e, ok := err.(interface{ Error() string }); ok {
		_ = e
	}
	writeJSON(w, status, APIError{Code: code, Message: msg})
}

func parseID(r *http.Request, name string) (int64, bool) {
	v := r.PathValue(name)
	if v == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
