package acceptance

import (
	"testing"

	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug04_PublishRequiresSameBatchFrozenVersion(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	svc := service.New(db)
	a, err := svc.CreateBatch("A", "one")
	if err != nil { t.Fatal(err) }
	b, err := svc.CreateBatch("B", "two")
	if err != nil { t.Fatal(err) }
	v, err := svc.CreateVersion(a.ID)
	if err != nil { t.Fatal(err) }
	if _, err := svc.FreezeVersion(v.ID); err != nil { t.Fatal(err) }
	if _, err := svc.PublishBatch(b.ID); err == nil { t.Fatal("batch B published using batch A's frozen version") }
}
