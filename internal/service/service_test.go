package service

import (
	"path/filepath"
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
	"task213-crownreview/internal/store"
)

func TestUploadBlockIsIdempotentAndSurvivesReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "crownreview.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(db)
	batch, err := svc.CreateBatch("idempotence", "service test")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	in := &pointcloud.BlockInput{
		TreeID:   "tree-1",
		CoordSys: "local-tree",
		Points:   []model.Point3{{X: 0, Y: 0, Z: 0}, {X: 0, Y: 0, Z: 1}},
	}
	first, err := svc.UploadBlock(batch.ID, in)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	second, err := svc.UploadBlock(batch.ID, &pointcloud.BlockInput{
		TreeID:   in.TreeID,
		CoordSys: in.CoordSys,
		Points:   []model.Point3{in.Points[1], in.Points[0]},
	})
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if second.ID != first.ID {
		db.Close()
		t.Fatalf("idempotent upload returned block %d, want %d", second.ID, first.ID)
	}
	blocks, err := svc.ListBlocks(batch.ID)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		db.Close()
		t.Fatalf("idempotent upload created %d rows, want 1", len(blocks))
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	recovered, err := New(reopened).GetBlock(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.PointCount != len(in.Points) || recovered.Status != model.BlockStatusPending {
		t.Fatalf("reopened block = count %d status %q, want count %d status %q", recovered.PointCount, recovered.Status, len(in.Points), model.BlockStatusPending)
	}
}
