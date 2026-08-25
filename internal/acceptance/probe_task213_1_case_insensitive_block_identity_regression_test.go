package acceptance

import (
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug01_CaseInsensitiveBlockIdentity(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	svc := service.New(db)
	b, err := svc.CreateBatch("identity", "case")
	if err != nil { t.Fatal(err) }
	pts := []model.Point3{{X: 0, Y: 0, Z: 0}, {X: 0, Y: 0, Z: 1}}
	a, err := svc.UploadBlock(b.ID, &pointcloud.BlockInput{TreeID: "Tree-1", CoordSys: "LOCAL-TREE", Points: pts})
	if err != nil { t.Fatal(err) }
	c, err := svc.UploadBlock(b.ID, &pointcloud.BlockInput{TreeID: "tree-1", CoordSys: "local-tree", Points: []model.Point3{pts[1], pts[0]}})
	if err != nil { t.Fatal(err) }
	if a.ID != c.ID { t.Fatalf("case/order variant created block %d, want existing block %d", c.ID, a.ID) }
}
