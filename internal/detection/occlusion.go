package detection

import (
	"math"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
)

// DetectOcclusion finds uncertain regions caused by sensor occlusion.
//
// A voxel with very few points but surrounded by denser voxels indicates the
// beam was blocked (e.g. by foliage), producing an uncertain reconstruction zone.
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
		// count occupied 26-neighbors
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
			// sparse cell surrounded by dense cells => occlusion uncertainty
			c := grid.Centroid(k)
			score := 1 - float64(d)/float64(sparseThreshold)
			if score < 0 {
				score = 0
			}
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
