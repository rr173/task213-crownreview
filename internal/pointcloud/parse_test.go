package pointcloud

import (
	"testing"

	"task213-crownreview/internal/model"
)

func TestHashIdempotentAcrossOrder(t *testing.T) {
	a := []model.Point3{{X: 1, Y: 2, Z: 3}, {X: 4, Y: 5, Z: 6}}
	b := []model.Point3{{X: 4, Y: 5, Z: 6}, {X: 1, Y: 2, Z: 3}}
	if Hash("t1", "local-tree", a) != Hash("t1", "local-tree", b) {
		t.Fatalf("hash should be order-independent")
	}
	if Hash("t1", "local-tree", a) == Hash("t2", "local-tree", a) {
		t.Fatalf("hash should differ by tree id")
	}
}

func TestValidateRejectsUnknownCoord(t *testing.T) {
	in := &BlockInput{TreeID: "t1", CoordSys: "bogus", Points: []model.Point3{{X: 1}}}
	if err := in.Validate(); err == nil {
		t.Fatalf("expected unknown coord sys to be rejected")
	}
}

func TestBoundingBox(t *testing.T) {
	pts := []model.Point3{{X: -1, Y: 0, Z: 2}, {X: 3, Y: 4, Z: -5}}
	min, max := BoundingBox(pts)
	if min != [3]float64{-1, 0, -5} || max != [3]float64{3, 4, 2} {
		t.Fatalf("unexpected bbox min=%v max=%v", min, max)
	}
}
