package acceptance

import (
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug03_FreezeRejectsOpenCandidate(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	svc := service.New(db)
	b, err := svc.CreateBatch("freeze", "guard")
	if err != nil { t.Fatal(err) }
	blk, err := svc.UploadBlock(b.ID, &pointcloud.BlockInput{TreeID: "T1", CoordSys: "local-tree", Points: []model.Point3{{Z: 0}, {Z: 1}}})
	if err != nil { t.Fatal(err) }
	if _, err := db.CreateCandidate(&model.BreakCandidate{BlockID: blk.ID, TreeID: "T1", Position: [3]float64{0, 0, .5}, Severity: .2, Confidence: .5}); err != nil { t.Fatal(err) }
	v, err := svc.CreateVersion(b.ID)
	if err != nil { t.Fatal(err) }
	if _, err := svc.FreezeVersion(v.ID); err == nil { t.Fatal("freeze succeeded with an open candidate") }
}
