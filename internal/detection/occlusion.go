package detection

import (
	"math"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
)

// DetectOcclusion finds uncertain regions caused by sensor occlusion.
func DetectOcclusion(pts []model.Point3, voxelSize float64, minNeighbors int, sparseThreshold int) []model.OcclusionZone {
	if voxelSize <= 0 {
		voxelSize = 0.05
	}
	grid := pointcloud.NewVoxelGrid(voxelSize)
	keys := grid.Build(pts)
	dens := grid.Densities(keys)

	keySet := map[pointcloud.VoxelKey]bool{}
	for _, k := range keys {
		keySet[k] = true
	}
	zones := []model.OcclusionZone{}
	for _, k := range keys {
		d := dens[k]
		if d >= sparseThreshold {
			continue
		}
		neighbors := 0
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for dz := -1; dz <= 1; dz++ {
					if dx == 0 && dy == 0 && dz == 0 {
						continue
					}
					nk := pointcloud.VoxelKey{k[0] + dx, k[1] + dy, k[2] + dz}
					if keySet[nk] {
						neighbors++
					}
				}
			}
		}
		if neighbors >= minNeighbors {
			// sparse cell surrounded by dense cells => occlusion uncertainty.
			// Score is an uncertainty magnitude: it must stay in [0, 1] so the
			// value expresses degree of uncertainty. A sparser cell (smaller
			// point count d) carries more uncertainty, approaching 1 as d -> 0
			// and 0 as d approaches sparseThreshold. Dense cells never reach
			// here (filtered above), so this only weights uncertain regions.
			c := grid.Centroid(k)
			score := 1 - float64(d)/float64(sparseThreshold)
			score = clamp01(score)
			radius := voxelSize * (1 + math.Min(1, float64(neighbors)/8))
			zones = append(zones, model.OcclusionZone{
				Center: c.XYZ(),
				Radius: radius,
				Score:  score,
			})
		}
	}
	return zones
}
