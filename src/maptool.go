package main

import "flag"
import "fmt"
import "io"
import "log"
import "os"
import "runtime"
import "database/sql"
import "io/ioutil"

import "common"
import "data"
import "geo"
import "graph"
import "maps"

var input = flag.String("input", "", "OSM PBF formatted file")
var contraction_program = flag.String("contraction_program",
	"../bin/contraction", "Program for computing ch-format")
var tmp_dir = flag.String("tmp_dir",
	"../bin/contraction", "Program for computing ch-format")

var highwayTypes = map[string]bool{
	"motorway":       true,
	"motorway_link":  true,
	"trunk":          true,
	"trunk_link":     true,
	"primary":        true,
	"primary_link":   true,
	"secondary":      true,
	"secondary_link": true,
	"tertiary":       false,
	"tertiary_link":  false,
	"living_street":  false,
	"residential":    false,
	"unclassified":   false,
	"service":        false,
	"road":           false,
}

type mapId int64

type mapCount struct {
	id graph.NodeId
	ec uint32  // edge count
}

type mapData1 struct {
	// The set of node IDs we wish to keep during the following
	// scan and renumberings into a (32-bit dense ID number,
	// 32-bit count of outgoing edges)
	mapIds     map[mapId]mapCount
	nextNodeId graph.NodeId
	totalEdges uint32
}

type node struct {
	id                  graph.NodeId
	position            [3]geo.EarthLoc
	treeLeft, treeRight graph.NodeId
	neighbors           []graph.NodeId
}

type mapData2 struct {
	nodes []node
	edges []graph.NodeId
}

type nodeDist struct {
	id graph.NodeId
	dist float64
}

type mapTool struct {
	data.ConvoyData
	loc2node map[common.CityState]nodeDist
	tree *geo.Tree
	data *mapData2
	input *mapData1
}

type cityNode struct {
	cs common.CityState
	nd nodeDist
}

type cityPair struct {
	from, to common.CityState
	fromNodeD, toNodeD nodeDist
}

type cityDist struct {
	from, to common.CityState
	meters int
}

func keepWay(way *maps.Way) bool {
	for _, a := range way.Attrs {
		if a.Key == "highway" {
			if yes, has := highwayTypes[a.Value]; has {
				return yes
			}
		}
	}
	return false
}

func (md *mapData1) mapPass1(bd *maps.BlockData) {
	for w := 0; w < len(bd.Ways); w++ {
		way := &bd.Ways[w]
		if !keepWay(way) {
			continue
		}
		if len(way.Refs) < 2 {
			continue
		}
		md.totalEdges += uint32(len(way.Refs)) - 1
		for e, ref := range way.Refs {
			var edges uint32
			if e == 0 || e == (len(way.Refs)-1) {
				edges = 1
			} else {
				edges = 2
			}

			value, has := md.mapIds[mapId(ref)]
			if has {
				value.ec += edges
				md.mapIds[mapId(ref)] = value
				continue
			}

			md.mapIds[mapId(ref)] = mapCount{md.nextNodeId, edges}
			md.nextNodeId++
		}
	}
}

func (md *mapData2) addEdge(n0 *node, v1 graph.NodeId) {
	for i, neighbor := range n0.neighbors {
		if neighbor != graph.ZeroNodeId {
			continue
		}
		n0.neighbors[i] = v1
		return
	}
	panic("Invalid edge count")
}

func (md *mapData2) addEdges(v0, v1 graph.NodeId) {
	md.addEdge(&md.nodes[v0], v1)
	md.addEdge(&md.nodes[v1], v0)
}

func (md *mapData2) mapPass2(bd *maps.BlockData, md1 *mapData1) {
	for n := 0; n < len(bd.Nodes); n++ {
		mapnode := &bd.Nodes[n]
		mc, has := md1.mapIds[mapId(mapnode.Id)]
		if !has {
			continue
		}
		mn := &md.nodes[mc.id]
		mn.id = mc.id
		geo.SphereCoords{Lat: mapnode.Lat, Long: mapnode.Long}.
			ToCoords(mn.position[:])
	}
	for w := 0; w < len(bd.Ways); w++ {
		way := &bd.Ways[w]
		if !keepWay(way) {
			continue
		}
		if len(way.Refs) < 2 {
			continue
		}
		for e := 1; e < len(way.Refs); e++ {
			mc0, has0 := md1.mapIds[mapId(way.Refs[e-1])]
			mc1, has1 := md1.mapIds[mapId(way.Refs[e])]
			if !has0 || !has1 {
				panic("Corrupted mapIds?")
			}
			md.addEdges(mc0.id, mc1.id)
		}
	}
}

func readInput() io.Reader {
	file, err := os.Open(*input)
	if err != nil {
		log.Fatalln("Could not open file:", *input, ":", err)
	}
	log.Println("Reading", *input)
	return file
}

func newMapData1() *mapData1 {
	return &mapData1{
		mapIds:     make(map[mapId]mapCount),
		nextNodeId: graph.FirstNodeId,
		totalEdges: 0,
	}
}

func newMapData2(md1 *mapData1) *mapData2 {
	md2 := &mapData2{
		make([]node, md1.nextNodeId),
		make([]graph.NodeId, md1.totalEdges*2),
	}
	ei := 0
	c := 0
	for _, mc := range md1.mapIds {
		np := &md2.nodes[mc.id]
		np.neighbors = md2.edges[ei : ei+int(mc.ec)]
		ei += int(mc.ec)
		c++
	}
	if ei != len(md2.edges) {
		panic(fmt.Sprintln("Incorrect edge count", ei, len(md2.edges)))
	}
	return md2
}

func main() {
	data.Main(programBody)
}

func programBody(db *sql.DB) error {
	var mt mapTool
	cd, err := data.NewConvoyData(db)
	if err != nil {
		return err
	}
	mt.ConvoyData = *cd
	mt.loc2node = make(map[common.CityState]nodeDist)

	osm := maps.NewMap()

	md1 := newMapData1()
	if err := osm.ReadMap(readInput(), func(bd *maps.BlockData) {
		md1.mapPass1(bd)
	}); err != nil {
		return err
	}
	common.PrintMem()
	log.Println("Using", md1.nextNodeId, "nodes, have",
		md1.totalEdges, "edges")

	md2 := newMapData2(md1)
	if err := osm.ReadMap(readInput(), func(bd *maps.BlockData) {
		md2.mapPass2(bd, md1)
	}); err != nil {
		return err
	}
	common.PrintMem()
	mt.input = md1

	// Sanity check: should have filled-in all edges
	for _, e := range md2.edges {
		if e == graph.ZeroNodeId {
			panic("Did not fill-in all edges")
		}
	}
	mt.data = md2
	mt.tree = geo.NewTree(md2)
	mt.tree.Build()
	log.Println("Built geospatial tree")
	common.PrintMem()

	if err := mt.findCityNodes(); err != nil {
		return err
	}		

	// if err := mt.findCityDistances(); err != nil {
	// 	return err
	// }

	ddsgName, err := mt.writeDdsg()
	if err != nil {
		return err
	}
	log.Println("Wrote ddsg file:", ddsgName)
	return nil
}

func (n *node) Point() geo.Coords {
	return n.position[:]
}

func (n *node) Left(g geo.Graph) geo.Vertex {
	if n.treeLeft == 0 {
		return nil
	}
	return &g.(*mapData2).nodes[n.treeLeft]
}

func (n *node) Right(g geo.Graph) geo.Vertex {
	if n.treeRight == 0 {
		return nil
	}
	return &g.(*mapData2).nodes[n.treeRight]
}

func (n *node) SetLeft(g geo.Graph, v geo.Vertex) {
	if v != nil {
		n.treeLeft = v.(*node).id
	}
}

func (n *node) SetRight(g geo.Graph, v geo.Vertex) {
	if v != nil {
		n.treeRight = v.(*node).id
	}
}

func (n *node) String() string {
	return fmt.Sprintf("(%v:%v,%v,%v)",
		n.id, n.position[0], n.position[1], n.position[2])
}

func (md *mapData2) Count() int {
	return len(md.nodes) - 1
}

func (md *mapData2) Node(i int) geo.Vertex {
	return &md.nodes[i+1]
}

func (md *mapData2) Edges() int {
	return len(md.edges) / 2
}

func (md *mapData2) Neighbors(n graph.NodeId) []graph.NodeId {
	return md.nodes[n].neighbors
}

func (md *mapData2) Weight(from, to graph.NodeId) float64 {
	return geo.GreatCircleDistance(
		md.nodes[from].position[:], md.nodes[to].position[:])
}

func (mt *mapTool) locateCity(csl geo.CityStateLoc) nodeDist {
	var coords [3]geo.EarthLoc
	csl.SphereCoords.ToCoords(coords[:])
	near := mt.tree.FindNearest(coords[:])
	dist := geo.GreatCircleDistance(near.Point(), coords[:])
	log.Printf("%v @ %v nearest %.2fkm", 
		csl.CityState, csl.SphereCoords, dist / 1000.0)	
	return nodeDist{near.(*node).id, dist}
}

func (mt *mapTool) findCityNodes() error {
	cpus := runtime.NumCPU()
	ch1 := make(chan geo.CityStateLoc, cpus)
	ch2 := make(chan cityNode, cpus)
	ch3 := make(chan bool, cpus)
	for i := 0; i < cpus; i++ {
		go func() {
			for csl := range ch1 {
				ch2 <- cityNode{csl.CityState, mt.locateCity(csl)}
			}
			ch3 <- true
		}()
	}
	go func() {
		for csn := range ch2 {
			mt.loc2node[csn.cs] = csn.nd
		}
		ch3 <- true
	}()
	if err := mt.ForAllLocations(func (_ int64, csl geo.CityStateLoc) error {
		ch1 <- csl
		return nil
	}); err != nil {
		return err
	}
	close(ch1)
	for i := 0; i < cpus; i++ {
		<- ch3
	}
	close(ch2)
	<- ch3
	return nil
}

func (mt *mapTool) shortestPath(csp cityPair) int {
	nodes := graph.ShortestPath(mt.data, csp.fromNodeD.id, csp.toNodeD.id)
	var dist float64
	for i := 0; i < len(nodes) - 1; i++ {
		dist += mt.data.Weight(nodes[i], nodes[i+1])
	}
	dist += csp.fromNodeD.dist
	dist += csp.toNodeD.dist
	fromP := mt.data.nodes[csp.fromNodeD.id].Point()
	toP := mt.data.nodes[csp.toNodeD.id].Point()
	log.Printf("%v -> %v = %.1fkm (%.1f%%) %d segments",
		csp.from, csp.to, dist / 1000.0, 
		100.0 * (float64(dist) / geo.GreatCircleDistance(fromP, toP)),
		len(nodes))
	return int(dist)
}

func (mt *mapTool) findCityDistances() error {
	cpus := runtime.NumCPU()
	ch1 := make(chan cityPair, cpus)
	ch2 := make(chan cityDist, cpus)
	ch3 := make(chan bool, cpus)
	for i := 0; i < cpus; i++ {
		go func() {
			for csp := range ch1 {
				ch2 <- cityDist{csp.from, csp.to, mt.shortestPath(csp)}
			}
			ch3 <- true
		}()
	}
	go func() {
		for csd := range ch2 {
			if err := mt.AddRoadDistance(csd.from, csd.to, csd.meters / 1000); err != nil {
				log.Println("AddRoadDistance", csd.from, csd.to,
					"failed:", err)
			}
		}
		ch3 <- true
	}()
	if err := mt.ForAllLoadPairsMissingDistance(
		func (from, to geo.CityStateLoc) error {

		fromNodeD, has1 := mt.loc2node[from.CityState]
		toNodeD, has2 := mt.loc2node[to.CityState]
		
		if !has1 || !has2 {
			log.Println("Missing a location:", from, to)
			return nil
		}

		ch1 <- cityPair{from.CityState, to.CityState, fromNodeD, toNodeD}
		return nil
	}); err != nil {
		return err
	}
	close(ch1)
	for i := 0; i < cpus; i++ {
		<- ch3
	}
	close(ch2)
	<- ch3
	return nil
}

func (mt *mapTool) writeDdsg() (string, error) {
	f, err := ioutil.TempFile("", "map_ddsg")
	// Compute a graph of only road intersections and source/dest locations.
	keep := make(map[graph.NodeId]bool)

	for _, nd := range mt.loc2node {
		keep[nd.id] = true
	}
	for i := graph.FirstNodeId; i < graph.NodeId(len(mt.data.nodes)); i++ {
		if len(mt.data.nodes[i].neighbors) > 2 {
			keep[i] = true
		}
	}
	log.Println("Condensed graph keep", len(keep), "nodes")
	edges := graph.Condense(mt.data, func (n graph.NodeId) bool {
		return keep[n]
	})
	condensed := graph.EdgelistToGraph(edges)
	cedges := graph.GraphToEdgelist(condensed)
	err = graph.WriteDdsg(condensed.Count(), cedges, f)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return "", err
	}
	return name, nil
}
