package graph;

type construct struct {
	g Graph
	keepf func(n NodeId) bool
	busy map[NodeId]bool
	edges Edgelist
}

func Condense(g Graph, keepf func(n NodeId) bool) Edgelist {
	c := &construct{g: g, keepf: keepf, busy: make(map[NodeId]bool)}
	for n := FirstNodeId; n <= NodeId(g.Count()); n++ {
		if !c.keepf(n) {
			continue
		}
		c.condenseNode(n, n, 0.0)
	}
	return c.edges
}

func (c *construct) condenseNode(start, n NodeId, weight float64) {
	if c.busy[n] {
		return
	}
	c.busy[n] = true
	for _, nn := range c.g.Neighbors(n) {
		c.condenseEdge(start, n, nn, weight)
	}
	c.busy[n] = false
}

func (c *construct) condenseEdge(start, pos, end NodeId, accum float64) {
	accum += c.g.Weight(pos, end)
	if c.keepf(end) {
		if start < end {
			c.edges = append(c.edges, Edge{start, end, accum})
		}
		return
	}
	c.condenseNode(start, end, accum)
}

