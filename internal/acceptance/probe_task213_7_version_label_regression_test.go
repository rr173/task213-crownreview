package acceptance

import (
	"testing"

	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug07_VersionLabelSanitizesPathSeparators(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	svc := service.New(db)
	b, err := svc.CreateBatch("north/park tree", "label")
	if err != nil { t.Fatal(err) }
	v, err := svc.CreateVersion(b.ID)
	if err != nil { t.Fatal(err) }
	if v.Label != "north-park-tree-v0" { t.Fatalf("label = %q", v.Label) }
}
