package acceptance

import (
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/service"
	"task213-crownreview/internal/store"
)

func TestBug10_ReparseDoesNotDuplicateDerivedRows(t *testing.T) {
	db, err := store.Open("")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	svc := service.New(db)
	b, err := svc.CreateBatch("repeat", "parse")
	if err != nil { t.Fatal(err) }
	blk, err := svc.UploadBlock(b.ID, &pointcloud.BlockInput{TreeID: "T1", CoordSys: "local-tree", Points: []model.Point3{{Z: 0}, {Z: .04}, {Z: .08}}})
	if err != nil { t.Fatal(err) }
	if _, err := svc.ParseBlock(blk.ID); err != nil { t.Fatal(err) }
	firstEdges, err := svc.GetSkeleton(blk.ID)
	if err != nil { t.Fatal(err) }
	firstCandidates, err := svc.ListCandidates(blk.ID, "")
	if err != nil { t.Fatal(err) }
	if _, err := svc.ParseBlock(blk.ID); err != nil { t.Fatal(err) }
	secondEdges, err := svc.GetSkeleton(blk.ID)
	if err != nil { t.Fatal(err) }
	secondCandidates, err := svc.ListCandidates(blk.ID, "")
	if err != nil { t.Fatal(err) }
	if len(firstEdges) != len(secondEdges) || len(firstCandidates) != len(secondCandidates) { t.Fatalf("reparse changed derived counts: edges %d/%d candidates %d/%d", len(firstEdges), len(secondEdges), len(firstCandidates), len(secondCandidates)) }
}
