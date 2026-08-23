package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestHandlerServesHealthWebAndCreateBatch(t *testing.T) {
	db, err := store.Open("")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := NewServer(service.New(db), db, ":0", "").Handler()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK || !strings.Contains(health.Body.String(), `"status":"ok"`) {
		t.Fatalf("health response = %d %s", health.Code, health.Body.String())
	}

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "城市树冠激光点云断枝复核台") {
		t.Fatalf("web response = %d, body missing title", page.Code)
	}

	created := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/batches", strings.NewReader(`{"name":"route-batch","description":"api test"}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(created, req)
	if created.Code != http.StatusCreated {
		t.Fatalf("create batch status = %d, body=%s", created.Code, created.Body.String())
	}
	var body struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "route-batch" || body.Status != "uploading" {
		t.Fatalf("created batch = %+v", body)
	}
}
