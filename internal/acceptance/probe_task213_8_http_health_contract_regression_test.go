package acceptance

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task213-crownreview/internal/httpapi"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug08_HealthResponseHasJSONContract(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	rr := httptest.NewRecorder()
	httpapi.NewServer(service.New(db), db, ":0", "").Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rr.Code != http.StatusOK { t.Fatalf("status = %d", rr.Code) }
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/json") { t.Fatalf("content type = %q", rr.Header().Get("Content-Type")) }
	if !strings.Contains(rr.Body.String(), `"module":"task213-crownreview"`) { t.Fatalf("body = %s", rr.Body.String()) }
}
