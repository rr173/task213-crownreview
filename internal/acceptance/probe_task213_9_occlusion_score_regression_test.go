package acceptance

import (
	"testing"

	"task213-crownreview/internal/detection"
	"task213-crownreview/internal/model"
)

func TestBug09_OcclusionScoresStayBounded(t *testing.T) {
	pts := []model.Point3{{X: 0, Y: 0, Z: 0}, {X: .05, Y: 0, Z: 0}, {X: -.05, Y: 0, Z: 0}, {X: 0, Y: .05, Z: 0}, {X: 0, Y: -.05, Z: 0}}
	for _, zone := range detection.DetectOcclusion(pts, .05, 1, 3) {
		if zone.Score < 0 || zone.Score > 1 { t.Fatalf("occlusion score %v out of [0,1]", zone.Score) }
	}
}
