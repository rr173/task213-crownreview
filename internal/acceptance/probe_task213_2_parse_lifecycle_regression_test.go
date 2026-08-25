package acceptance

import (
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug02_ParseCompletesLifecycle(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	svc := service.New(db)
	b, err := svc.CreateBatch("parse", "lifecycle")
	if err != nil { t.Fatal(err) }
	blk, err := svc.UploadBlock(b.ID, &pointcloud.BlockInput{TreeID: "T1", CoordSys: "local-tree", Points: []model.Point3{{Z: 0}, {Z: 1}}})
	if err != nil { t.Fatal(err) }
	parsed, err := svc.ParseBlock(blk.ID)
	if err != nil { t.Fatal(err) }
	if parsed.Status != model.BlockStatusLayered { t.Fatalf("block status = %q, want layered", parsed.Status) }
	got, err := svc.GetBatch(b.ID)
	if err != nil { t.Fatal(err) }
	if got.Status != model.BatchStatusReviewing { t.Fatalf("batch status = %q, want reviewing", got.Status) }
}
