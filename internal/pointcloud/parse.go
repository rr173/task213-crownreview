package pointcloud

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"task213-crownreview/internal/model"
)

// Known coordinate systems accepted on upload.
var KnownCoordSystems = map[string]bool{
	"local-tree":  true,
	"epsgs:4549":  true,
	"epsgs:4978":  true,
	"enu-local":   true,
}

// BlockInput is the raw payload for one point cloud block.
type BlockInput struct {
	TreeID   string
	CoordSys string
	Points   []model.Point3
}

// Validate checks format and coordinate system.
func (in *BlockInput) Validate() error {
	if strings.TrimSpace(in.TreeID) == "" {
		return model.Errf(model.ErrTreeIDConflict, "empty tree id")
	}
	if !KnownCoordSystems[strings.ToLower(strings.TrimSpace(in.CoordSys))] {
		return model.Errf(model.ErrUnknownCoordSys, "coord_sys=%q", in.CoordSys)
	}
	if len(in.Points) == 0 {
		return model.Errf(model.ErrInvalidArgument, "no points")
	}
	for i, p := range in.Points {
		if p.X != p.X || p.Y != p.Y || p.Z != p.Z {
			return model.Errf(model.ErrBadFormat, "point %d has NaN coordinate", i)
		}
	}
	return nil
}

// BoundingBox computes the axis-aligned bounding box of the points.
func BoundingBox(pts []model.Point3) ([3]float64, [3]float64) {
	if len(pts) == 0 {
		return [3]float64{0, 0, 0}, [3]float64{0, 0, 0}
	}
	min := [3]float64{pts[0].X, pts[0].Y, pts[0].Z}
	max := min
	for _, p := range pts[1:] {
		v := [3]float64{p.X, p.Y, p.Z}
		for k := 0; k < 3; k++ {
			if v[k] < min[k] {
				min[k] = v[k]
			}
			if v[k] > max[k] {
				max[k] = v[k]
			}
		}
	}
	return min, max
}

// Hash derives an idempotency key from tree id, coord system and normalized points.
func Hash(treeID, coordSys string, pts []model.Point3) string {
	h := sha256.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(treeID))))
	h.Write([]byte("|"))
	h.Write([]byte(strings.ToLower(strings.TrimSpace(coordSys))))
	h.Write([]byte("|"))
	// Sort a copy so point order does not affect the hash (idempotent across uploads).
	cp := make([]model.Point3, len(pts))
	copy(cp, pts)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].X != cp[j].X {
			return cp[i].X < cp[j].X
		}
		if cp[i].Y != cp[j].Y {
			return cp[i].Y < cp[j].Y
		}
		return cp[i].Z < cp[j].Z
	})
	for _, p := range cp {
		fmt.Fprintf(h, "%.6f,%.6f,%.6f,%.4f;", p.X, p.Y, p.Z, p.Intensity)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Summary builds a short human-readable summary of a block.
func Summary(treeID string, pts []model.Point3) string {
	min, max := BoundingBox(pts)
	return fmt.Sprintf("tree=%s points=%d bbox=[%.2f,%.2f,%.2f]-[%.2f,%.2f,%.2f]",
		treeID, len(pts), min[0], min[1], min[2], max[0], max[1], max[2])
}
