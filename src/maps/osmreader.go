package maps

import "bytes"
import "compress/zlib"
import "encoding/binary"
import "errors"
import "fmt"
import "io"
import "io/ioutil"
import "log"
import "runtime"

import "code.google.com/p/goprotobuf/proto"

import "proto/osm"
import "geo"

var (
	numReaderProcs = runtime.NumCPU() - 1
)

type Map struct {
	Nodes map[int64]*Node
	Ways map[int64]*Way
	Rels map[int64]*Relation
	Tree *geo.Tree
	blockCh chan *osm.Blob
	graphCh chan *blockData
	doneCh chan bool
}

type blockData struct {
	nodes []Node
	ways []Way
	rels []Relation
}

type blockParams struct {
	strings [][]byte
	granularity int64
	latOffset int64
	lonOffset int64
}

type MemberType int

const (
    NODE = 0
    WAY = 1
    RELATION = 2
)

type Attribute struct {
	Key, Value string
}

type Node struct {
	Id int64
	Point [2]geo.ScaledRad  // (Lat, Lon)
	Attrs []Attribute
}

type Way struct {
	Id int64
	Attrs []Attribute
	Refs []int64
}

type RelEntry struct {
	Member int64
	Type MemberType
	Role string
}

type Relation struct {
	Id int64
	Attrs []Attribute
	Ents []RelEntry
}

func NewMap() *Map {
	return &Map{
		make(map[int64]*Node),
		make(map[int64]*Way),
		make(map[int64]*Relation),
		geo.NewTree(2),
		make(chan *osm.Blob),
		make(chan *blockData),
		make(chan bool),
	}
}

func readFixed(f io.Reader, s int32) ([]byte, error) {
	buf, err := ioutil.ReadAll(&io.LimitedReader{f, int64(s)})
	if err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return nil, io.EOF
	}
	if len(buf) != int(s) {
		return nil, errors.New(
			fmt.Sprintln("Insufficient read:", len(buf), s))
	}
	return buf, nil
}

func decodeDenseNodes(dn *osm.DenseNodes, bp *blockParams) ([]Node, error) {
	ids := dn.GetId()
	lats := dn.GetLat()
	lons := dn.GetLon()
	kvs := dn.GetKeysVals()
	if len(ids) != len(lats) || len(ids) != len(lons) {
		return nil, errors.New(fmt.Sprintf(
			"Incorrect DB lengths: %d %d %d",
			len(ids), len(lats), len(lons)))
	}
	nodes := make([]Node, len(ids))

	// Initial delta-encoding last-values
	var lid int64
	var llat int64
	var llon int64
	kvi := 0
	for i := 0; i < len(ids); i++ {
		lid += ids[i]
		llat += lats[i]
		llon += lons[i]
		n := &nodes[i]
		n.Id = lid
		n.Point[0] = geo.ScaleDegrees(1e-9 * 
			float64(bp.latOffset + (bp.granularity * llat)))
		n.Point[1] = geo.ScaleDegrees(1e-9 * 
			float64(bp.lonOffset + (bp.granularity * llon)))
		if kvi < len(kvs) {
			for kvi < len(kvs) && kvs[kvi] != 0 {
				n.Attrs = append(n.Attrs, 
					Attribute{string(bp.strings[kvs[kvi]]), 
					          string(bp.strings[kvs[kvi+1]])})
				kvi += 2
			}
			kvi++
		}
	}
	return nodes, nil
}

func decodeWay(pway *osm.Way, way *Way, bp *blockParams) error {
	way.Id = pway.GetId()
	way.Attrs = decodeAttrs(pway.GetKeys(), pway.GetVals(), bp)
	way.Refs = make([]int64, len(pway.GetRefs()))
	var lref int64
	for i, dref := range pway.GetRefs() {
		lref += dref
		way.Refs[i] = lref
	}
	return nil
}

func decodeAttrs(keys, vals []uint32, bp *blockParams) []Attribute {
	attrs := make([]Attribute, len(keys))
	for i := 0; i < len(keys); i++ {
		attrs[i].Key = string(bp.strings[keys[i]])
		attrs[i].Value = string(bp.strings[vals[i]])
	}
	return attrs
}

func decodeRelation(prel *osm.Relation, rel *Relation, bp *blockParams) error {
	rel.Id = prel.GetId()
	rel.Attrs = decodeAttrs(prel.GetKeys(), prel.GetVals(), bp)
	rel.Ents = make([]RelEntry, len(prel.GetMemids()))
	var lmemid int64
	for i, dmemid := range prel.GetMemids() {
		lmemid += dmemid
		rel.Ents[i].Member = lmemid
		rel.Ents[i].Role = string(bp.strings[prel.GetRolesSid()[i]])
		rel.Ents[i].Type = MemberType(prel.GetTypes()[i])
	}
	return nil
	
}

func decodeWays(pways []*osm.Way, bp *blockParams) ([]Way, error) {
	ways := make([]Way, len(pways))
	for i, pway := range pways {
		if err := decodeWay(pway, &ways[i], bp); err != nil {
			return nil, err
		}
	}
	return ways, nil
}

func decodeRelations(prels []*osm.Relation, bp *blockParams) ([]Relation, error) {
	rels := make([]Relation, len(prels))
	for i, prel := range prels {
		if err := decodeRelation(prel, &rels[i], bp); err != nil {
			return nil, err
		}
	}
	return rels, nil
}

func decodeBlock(pb *osm.PrimitiveBlock) (*blockData, error) {
	bparams := &blockParams{
		pb.GetStringtable().GetS(), 
		int64(pb.GetGranularity()),
		pb.GetLatOffset(), 
		pb.GetLonOffset() }
	bdata := &blockData{}
	for _, pg := range pb.GetPrimitivegroup() {
		for _, _ = range pg.GetNodes() {
			return nil, errors.New("Unexpected non-dense node!")
		}
		if dn := pg.GetDense(); dn != nil {
			nodes, err := decodeDenseNodes(dn, bparams)
			if err != nil {
				return nil, err
			}
			bdata.nodes = nodes
		}
		ways, err := decodeWays(pg.GetWays(), bparams)
		if err != nil {
			return nil, err
		}
		bdata.ways = ways
		relations, err := decodeRelations(pg.GetRelations(), bparams)
		if err != nil {
			return nil, err
		}
		bdata.rels = relations
	}
	return bdata, nil
}

func (m *Map) processBlock(blob *osm.Blob) (*blockData, error) {
	data, err := decompressBlob(blob)
	if err != nil {
		return nil, err
	}
	priblock := &osm.PrimitiveBlock{}
	if err := proto.Unmarshal(data, priblock); err != nil {
		return nil, err
	}
	bd, err := decodeBlock(priblock)
	if err != nil {
		return nil, err
	}
	return bd, nil
}

func (m *Map) decodeBlockFunc() {
	for blob := range m.blockCh {
		if blob == nil {
			break
		}
		bd, err := m.processBlock(blob)
		if err != nil {
			log.Print("Block decode failed!")  // @@@ TODO(jmacd)
			continue
		}
		m.graphCh <- bd
	}
	m.graphCh <- nil
}

func (m *Map) buildGraph() {
	nils := 0
	for bd := range m.graphCh {
		if bd == nil {
			nils++
			if nils == numReaderProcs {
				break
			}
			continue
		}
		for i := 0; i < len(bd.nodes); i++ {
			n := &bd.nodes[i]
			m.Nodes[n.Id] = n
		}
		for i := 0; i < len(bd.ways); i++ {
			w := &bd.ways[i]
			m.Ways[w.Id] = w
		}
		for i := 0; i < len(bd.rels); i++ {
			r := &bd.rels[i]
			m.Rels[r.Id] = r
		}
	}
	m.doneCh <- true
}

func decompressBlob(blob *osm.Blob) ([]byte, error) {
	var data []byte
	enc := "unknown"

	// Uncompress the raw data, if necessary
	switch {
	case blob.Raw != nil:
		enc = "raw"
		data = blob.Raw
	case blob.ZlibData != nil:
		enc = "zlib"
		zr, err := zlib.NewReader(bytes.NewReader(blob.ZlibData))
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		if data, err = readFixed(zr, blob.GetRawSize()); err != nil {
			return nil, err
		}
	case blob.LzmaData != nil:
		enc = "lzma"
	}
	if data == nil {
		return nil, errors.New("Unsupported OSM data encoding: " + enc)
	}
	return data, nil
}

func readHeader(b *osm.Blob) error {
	var hdrblock osm.HeaderBlock
	data, err := decompressBlob(b)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(data, &hdrblock); err != nil {
		return err
	}
	haveVersion := false
	haveDense := false
	for _, rf := range hdrblock.RequiredFeatures {
		switch rf {
		case "OsmSchema-V0.6":
			haveVersion = true
		case "DenseNodes":
			haveDense = true
		default:
			return errors.New("Unknown map required feature:" + rf);
		}
	}
	if !haveVersion || !haveDense {
		return errors.New("Unsupported map type: " + 
			proto.CompactTextString(&hdrblock))
	}
	return nil
}

func (m *Map) ReadMap(f io.Reader) error {
	var nread int64
	for i := 0; i < numReaderProcs; i++ {
		go m.decodeBlockFunc()
	}
	go m.buildGraph()
	for {
		// Read the next blob header size
		hsizeb, err := readFixed(f, 4)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		nread += int64(4)

		// Decode four bytes
		var hsize int32
		binary.Read(bytes.NewReader(hsizeb), binary.BigEndian, &hsize)

		// Read the next blob header
		headb, err := readFixed(f, hsize)
		if err != nil {
			return err
		}
		nread += int64(hsize)

		// Unmarshal the header
		var bh osm.BlobHeader
		if err = proto.Unmarshal(headb, &bh); err != nil {
			return err
		}
		
		// Read the blob itself
		bsize := bh.GetDatasize()
		if bsize <= 0 {
			return errors.New("Zero byte blob; quitting")
		}
		blobb, err := readFixed(f, bsize)
		if err != nil {
			return err
		}
		nread += int64(bsize)
		
		// Unmarshal the blob
		blob := &osm.Blob{}
		if err = proto.Unmarshal(blobb, blob); err != nil {
			return err
		}

		// Now process each blob
		switch bh.GetType() {
		case "OSMHeader":
			if err := readHeader(blob); err != nil {
				return err
			}
		case "OSMData":
			m.blockCh <- blob
		default:
			return errors.New("Unknown OSM blob type: " + 
				bh.GetType())
		}
	}
	for i := 0; i < numReaderProcs; i++ {
		m.blockCh <- nil
	}
	var _ = <- m.doneCh
	log.Println("Finished reading", nread, "bytes", 
		len(m.Nodes), "nodes",
		len(m.Ways), "ways",
		len(m.Rels), "relations")
	nodes := make([]geo.Node, len(m.Nodes))
	node_i := 0
	for _, node := range m.Nodes {
		nodes[node_i] = node
		node_i++
	}
	m.Tree.Build(nodes)
	// na := make(map[string]bool)
	// wa := make(map[string]bool)
	// ra := make(map[string]bool)
	// fu := func(s map[string]bool, as []Attribute) {
	// 	for _, a := range as {
	// 		ck := a.Key + "=" + a.Value
	// 		s[ck] = true
	// 	}
	// }
	// for _, n := range m.Nodes {
	// 	fu(na, n.Attrs)
	// }
	// for _, w := range m.Ways {
	// 	fu(wa, w.Attrs)
	// }
	// for _, r := range m.Rels {
	// 	fu(ra, r.Attrs)
	// }
	// for a, _ := range na {
	// 	fmt.Println("NODE ATTR", a)
	// }
	// for a, _ := range wa {
	// 	fmt.Println("WAY ATTR", a)
	// }
	// for a, _ := range ra {
	// 	fmt.Println("REL ATTR", a)
	// }
	return nil
}

func (n *Node) Coord() []geo.ScaledRad {
	return n.Point[0:2]
}