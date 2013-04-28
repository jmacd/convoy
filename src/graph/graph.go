package graph

type graphNode struct {
	neighbors []NodeId
	weights []float64
}

type graph struct {
	nodes []graphNode
}

type Edge struct {
	n0, n1 NodeId
	weight float64
}

type Edgelist []Edge

func newGraph() *graph {
	return &graph{}
}

func (g *graph) Count() int {
	return len(g.nodes)
}

func (g *graph) Neighbors(id NodeId) []NodeId {
	return g.nodes[id-1].neighbors
}

func (g *graph) Weight(from, to NodeId) float64 {
	for i, n := range g.nodes[from-1].neighbors {
		if n == to {
			return g.nodes[from-1].weights[i]
		}
	}
	panic("Not found")
}

func (g *graph) addNode() NodeId {
	n := FirstNodeId + NodeId(len(g.nodes))
	g.nodes = append(g.nodes, graphNode{})
	return n
}

func (g *graph) addEdge(from, to NodeId, weight float64) {
	addFrom := func (from, to NodeId, weight float64) {
		g.nodes[from-1].neighbors = append(g.nodes[from-1].neighbors, to)
		g.nodes[from-1].weights = append(g.nodes[from-1].weights, weight)
	}
	addFrom(from, to, weight)
	addFrom(to, from, weight)
}

