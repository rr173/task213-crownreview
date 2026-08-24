package acceptance

import (
	"path/filepath"
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug05_BlockPointsSurviveRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "restart.db")
	db, err := store.Open(dbPath)
	if err != nil { t.Fatal(err) }
	svc := service.New(db)
	b, err := svc.CreateBatch("restart", "points")
	if err != nil { t.Fatal(err) }
	pts := []model.Point3{{X: 1, Y: 2, Z: 3}, {X: 4, Y: 5, Z: 6}}
	blk, err := svc.UploadBlock(b.ID, &pointcloud.BlockInput{TreeID: "T1", CoordSys: "local-tree", Points: pts})
	if err != nil { t.Fatal(err) }
	if err := db.Close(); err != nil { t.Fatal(err) }
	reopened, err := store.Open(dbPath)
	if err != nil { t.Fatal(err) }
	defer reopened.Close()
	got, err := reopened.GetBlockPoints(blk.ID)
	if err != nil { t.Fatal(err) }
	if len(got) != len(pts) || got[1].Z != pts[1].Z { t.Fatalf("reopened points = %+v", got) }
}
