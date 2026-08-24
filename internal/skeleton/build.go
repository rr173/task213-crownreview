package skeleton

import (
	"fmt"
	"math"

	"task213-crownreview/internal/model"
	"task213-crownreview/internal/pointcloud"
)

// Node is a skeleton graph node (voxel centroid).
type Node struct {
	Key  pointcloud.VoxelKey
	Pt   model.Point3
	Deg  int
}

// Graph is the branch skeleton connectivity graph.
type Graph struct {
	Nodes   []Node
	Index   map[pointcloud.VoxelKey]int
	Adj     [][2]int // edge endpoint node indices
	Connect map[[2]int]bool
}

// BuildOptions controls skeleton construction.
type BuildOptions struct {
	VoxelSize      float64 // voxel edge length (m)
	ConnectRadius  float64 // max distance to connect two voxel centroids
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() BuildOptions {
	return BuildOptions{VoxelSize: 0.05, ConnectRadius: 0.18}
}

func dist(a, b model.Point3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Build constructs the skeleton graph from raw points.
// Connections link voxel centroids within ConnectRadius (26-neighborhood aware).
func Build(pts []model.Point3, opt BuildOptions) *Graph {
	if opt.ConnectRadius <= 0 {
		opt = DefaultOptions()
	}
	grid := pointcloud.NewVoxelGrid(opt.VoxelSize)
	keys := grid.Build(pts)

	g := &Graph{Index: map[pointcloud.VoxelKey]int{}}
	for i, k := range keys {
		g.Nodes = append(g.Nodes, Node{Key: k, Pt: grid.Centroid(k)})
		g.Index[k] = i
	}
	g.Connect = map[[2]int]bool{}
	for i := 0; i < len(g.Nodes); i++ {
		for j := i + 1; j < len(g.Nodes); j++ {
			d := dist(g.Nodes[i].Pt, g.Nodes[j].Pt)
			if d <= opt.ConnectRadius {
				g.Adj = append(g.Adj, [2]int{i, j})
				g.Connect[[2]int{i, j}] = true
				g.Connect[[2]int{j, i}] = true
				g.Nodes[i].Deg++
				g.Nodes[j].Deg++
			}
		}
	}
	return g
}

// Edges converts the graph to persistable skeleton edges with radius = distance/2.
func (g *Graph) Edges() []model.SkeletonEdge {
	out := make([]model.SkeletonEdge, 0, len(g.Adj))
	for _, e := range g.Adj {
		a, b := g.Nodes[e[0]], g.Nodes[e[1]]
		// node id labels keep determinism for downstream detection.
		fa := fmt.Sprintf("n%d", e[0])
		fb := fmt.Sprintf("n%d", e[1])
		out = append(out, model.SkeletonEdge{
			FromNode:  fa,
			ToNode:    fb,
			FromPoint: a.Pt.XYZ(),
			ToPoint:   b.Pt.XYZ(),
			Radius:    dist(a.Pt, b.Pt) / 2,
		})
	}
	return out
}

// NodeByLabel returns a node by its "n<idx>" label.
func (g *Graph) NodeByLabel(label string) (Node, bool) {
	var idx int
	if _, err := fmt.Sscanf(label, "n%d", &idx); err != nil {
		return Node{}, false
	}
	if idx < 0 || idx >= len(g.Nodes) {
		return Node{}, false
	}
	return g.Nodes[idx], true
}

// Endpoints returns indices of degree-1 (dangling) nodes.
func (g *Graph) Endpoints() []int {
	out := []int{}
	for i, n := range g.Nodes {
		if n.Deg <= 1 {
			out = append(out, i)
		}
	}
	return out
}

// Components returns connected components as slices of node indices.
func (g *Graph) Components() [][]int {
	adj := make([][]int, len(g.Nodes))
	for _, e := range g.Adj {
		adj[e[0]] = append(adj[e[0]], e[1])
		adj[e[1]] = append(adj[e[1]], e[0])
	}
	seen := make([]bool, len(g.Nodes))
	var comps [][]int
	for i := range g.Nodes {
		if seen[i] {
			continue
		}
		stack := []int{i}
		seen[i] = true
		comp := []int{}
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, cur)
			for _, nb := range adj[cur] {
				if !seen[nb] {
					seen[nb] = true
					stack = append(stack, nb)
				}
			}
		}
		comps = append(comps, comp)
	}
	return comps
}

// Connected reports whether node a and b are in the same component.
func (g *Graph) Connected(a, b int) bool { return g.Connect[[2]int{a, b}] }
