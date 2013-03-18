package geo

type Tree struct {
}

type Node interface {
	Coord() []ScaledRad
}

func NewTree(d int) *Tree {
	return &Tree{}
}

func (t *Tree) Build(nodes []Node) {
}