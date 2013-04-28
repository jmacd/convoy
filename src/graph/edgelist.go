package graph

func GraphToEdgelist(g Graph) Edgelist {
	var edges Edgelist
	s := make(map[NodeId]bool)
	for n := FirstNodeId; n <= NodeId(g.Count()); n++ {
		edgeNode(g, n, s, &edges)
	}
	return edges
}

func edgeNode(g Graph, n NodeId, s map[NodeId]bool, e *Edgelist) {
	if s[n] {
		return
	}
	s[n] = true
	for _, nn := range g.Neighbors(n) {
		if n >= nn {
			*e = append(*e, Edge{n, nn, g.Weight(n, nn)})
		}
		edgeNode(g, nn, s, e)
	}
}

func nodeNo(n NodeId, g *graph, m map[NodeId]NodeId) NodeId {
	if v, has := m[n]; has {
		return v
	}
	v := g.addNode()
	m[n] = v
	return v
}

func EdgelistToGraph(edges Edgelist) Graph {
	g := newGraph()
	m := make(map[NodeId]NodeId)
	for _, e := range edges {
		n1, n2 := nodeNo(e.n0, g, m), nodeNo(e.n1, g, m)
		g.addEdge(n1, n2, e.weight)
	}
	return g
}
