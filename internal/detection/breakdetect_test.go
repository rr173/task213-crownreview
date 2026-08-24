package detection

import (
	"math"
	"testing"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/skeleton"
)

func branchPoints(baseZ, height float64, gap bool) []model.Point3 {
	pts := []model.Point3{}
	step := 0.04
	for i := 0; i < int(height/step); i++ {
		z := baseZ + float64(i)*step
		if gap && z > baseZ+height*0.6 && z < baseZ+height*0.6+0.3 {
			continue
		}
		pts = append(pts, model.Point3{X: 0, Y: 0, Z: z, Intensity: 0.8})
	}
	return pts
}

func TestDetectBreaksFindsGap(t *testing.T) {
	// A continuous branch should produce no breaks.
	cont := branchPoints(0, 2.0, false)
	g := skeleton.Build(cont, skeleton.DefaultOptions())
	if got := DetectBreaks(g, DefaultDetectOptions()); len(got) != 0 {
		t.Fatalf("continuous branch produced %d breaks, want 0", len(got))
	}

	// A branch with a gap should produce exactly one break.
	gapped := branchPoints(0, 2.0, true)
	g2 := skeleton.Build(gapped, skeleton.DefaultOptions())
	got := DetectBreaks(g2, DefaultDetectOptions())
	if len(got) == 0 {
		t.Fatalf("gapped branch produced no breaks, want >=1")
	}
	if got[0].Severity < DefaultDetectOptions().ContinuityRadius {
		t.Fatalf("break severity %f below continuity radius", got[0].Severity)
	}
}

func TestOcclusionFindsSparseRegion(t *testing.T) {
	// dense grid with a sparse hole in the middle
	pts := []model.Point3{}
	for x := -10; x <= 10; x++ {
		for y := -10; y <= 10; y++ {
			for z := -10; z <= 10; z++ {
				// leave a 3x3x3 sparse hole around origin
				if math.Abs(float64(x)) < 1.5 && math.Abs(float64(y)) < 1.5 && math.Abs(float64(z)) < 1.5 {
					continue
				}
				pts = append(pts, model.Point3{X: float64(x) * 0.05, Y: float64(y) * 0.05, Z: float64(z) * 0.05})
			}
		}
	}
	// add a single sparse point inside the hole region
	pts = append(pts, model.Point3{X: 0, Y: 0, Z: 0})
	zones := DetectOcclusion(pts, 0.05, 3, 2)
	if len(zones) == 0 {
		t.Fatalf("expected at least one occlusion zone around the sparse hole")
	}
}
