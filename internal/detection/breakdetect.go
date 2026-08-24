package detection

import (
	"fmt"
	"math"
	"sort"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/skeleton"
)

// DetectOptions controls break detection sensitivity.
type DetectOptions struct {
	// BreakGap is the maximum gap between two branch segments that still suggests a break.
	BreakGap float64
	// ContinuityRadius is the expected max gap for a normally connected branch.
	ContinuityRadius float64
	// MinSeverity is the minimum gap (m) worth reporting as a break.
	MinSeverity float64
}

// DefaultDetectOptions returns sensible defaults.
func DefaultDetectOptions() DetectOptions {
	return DetectOptions{BreakGap: 0.5, ContinuityRadius: 0.18, MinSeverity: 0.05}
}

func dist3(a, b [3]float64) float64 {
	dx, dy, dz := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// DetectBreaks finds suspected branch breaks from a skeleton graph.
//
// A break is detected as the closest pair of nodes belonging to two different
// connected components whose separation d satisfies:
//
//	ContinuityRadius < d <= BreakGap
//
// This models a point cloud gap: two branch segments that ought to be one
// continuous branch but were split by missing points (e.g. occlusion or an
// actual snapped branch). The break is reported at the midpoint of the two
// nearest nodes, with severity = d and confidence growing with gap size.
func DetectBreaks(g *skeleton.Graph, opt DetectOptions) []model.BreakCandidate {
	if opt.BreakGap <= 0 {
		opt = DefaultDetectOptions()
	}
	comps := g.Components()
	if len(comps) <= 1 {
		return nil
	}
	type pair struct {
		a, b int
		d    float64
	}
	var hits []pair
	// For each pair of components, find their closest node pair.
	for i := 0; i < len(comps); i++ {
		for j := i + 1; j < len(comps); j++ {
			best := pair{d: math.MaxFloat64}
			for _, a := range comps[i] {
				pa := g.Nodes[a].Pt.XYZ()
				for _, b := range comps[j] {
					d := dist3(pa, g.Nodes[b].Pt.XYZ())
					if d < best.d {
						best = pair{a: a, b: b, d: d}
					}
				}
			}
			if best.d > opt.ContinuityRadius && best.d <= opt.BreakGap {
				hits = append(hits, best)
			}
		}
	}
	sort.Slice(hits, func(x, y int) bool { return hits[x].d > hits[y].d })
	cands := make([]model.BreakCandidate, 0, len(hits))
	for _, h := range hits {
		if h.d < opt.MinSeverity {
			continue
		}
		na, nb := g.Nodes[h.a], g.Nodes[h.b]
		mid := midpoint(na.Pt, nb.Pt)
		conf := (h.d - opt.ContinuityRadius) / (opt.BreakGap - opt.ContinuityRadius)
		cands = append(cands, model.BreakCandidate{
			Status:     model.CandStatusOpen,
			Position:   mid,
			Severity:   h.d,
			Confidence: clamp01(conf),
			EdgeA:      fmt.Sprintf("n%d", h.a),
			EdgeB:      fmt.Sprintf("n%d", h.b),
			Reason: fmt.Sprintf(
				"branch discontinuity gap=%.3fm expected<=%.3fm (disconnected segments)",
				h.d, opt.ContinuityRadius),
		})
	}
	return cands
}

func midpoint(a, b model.Point3) [3]float64 {
	return [3]float64{(a.X + b.X) / 2, (a.Y + b.Y) / 2, (a.Z + b.Z) / 2}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
