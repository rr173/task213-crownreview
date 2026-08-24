package pointcloud

import (
	"math"
	"sort"

	"task213-crownreview/internal/model"
)

// VoxelKey is a grid cell coordinate.
type VoxelKey [3]int

// VoxelGrid downsamples points into occupied voxel centroids.
// size is the voxel edge length in meters.
type VoxelGrid struct {
	Size  float64
	Cells map[VoxelKey]model.Point3
	Count map[VoxelKey]int
	Sum   map[VoxelKey]model.Point3
}

// NewVoxelGrid creates a grid with the given voxel size.
func NewVoxelGrid(size float64) *VoxelGrid {
	if size <= 0 {
		size = 0.05
	}
	return &VoxelGrid{
		Size:  size,
		Cells: map[VoxelKey]model.Point3{},
		Count: map[VoxelKey]int{},
		Sum:   map[VoxelKey]model.Point3{},
	}
}

func (g *VoxelGrid) keyOf(p model.Point3) VoxelKey {
	return VoxelKey{
		int(math.Floor(p.X / g.Size)),
		int(math.Floor(p.Y / g.Size)),
		int(math.Floor(p.Z / g.Size)),
	}
}

// Add inserts a point into its voxel (accumulating for centroid).
func (g *VoxelGrid) Add(p model.Point3) {
	k := g.keyOf(p)
	g.Count[k]++
	g.Sum[k] = model.Point3{
		X:         g.Sum[k].X + p.X,
		Y:         g.Sum[k].Y + p.Y,
		Z:         g.Sum[k].Z + p.Z,
		Intensity: g.Sum[k].Intensity + p.Intensity,
	}
	g.Cells[k] = model.Point3{}
}

// Centroid returns the centroid of a voxel's accumulated points.
func (g *VoxelGrid) Centroid(k VoxelKey) model.Point3 {
	n := float64(g.Count[k])
	if n == 0 {
		return model.Point3{}
	}
	return model.Point3{
		X:         g.Sum[k].X / n,
		Y:         g.Sum[k].Y / n,
		Z:         g.Sum[k].Z / n,
		Intensity: g.Sum[k].Intensity / n,
	}
}

// Build fills the grid from points and returns ordered voxel keys.
func (g *VoxelGrid) Build(pts []model.Point3) []VoxelKey {
	for _, p := range pts {
		g.Add(p)
	}
	keys := make([]VoxelKey, 0, len(g.Cells))
	for k := range g.Cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		if keys[i][1] != keys[j][1] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][2] < keys[j][2]
	})
	return keys
}

// Densities returns the per-voxel point count in the same key order.
func (g *VoxelGrid) Densities(keys []VoxelKey) map[VoxelKey]int {
	out := make(map[VoxelKey]int, len(keys))
	for _, k := range keys {
		out[k] = g.Count[k]
	}
	return out
}
