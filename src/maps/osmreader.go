package maps

import "bytes"
import "compress/zlib"
import "encoding/binary"
import "errors"
import "fmt"
import "io"
import "io/ioutil"
import "runtime"
import "log"
import "code.google.com/p/goprotobuf/proto"

import "proto/osm"

var (
	numReaderProcs = runtime.NumCPU() - 1
)

type Map struct {
	Nodes map[int64]*Node
	blockCh chan *osm.PrimitiveBlock
	graphCh chan *blockData
	doneCh chan bool
}

type blockData struct {
	nodes []Node
}

type blockParams struct {
	strings [][]byte
	granularity int64
	latOffset int64
	lonOffset int64
}

type Attribute struct {
	Key, Value string
}

type Node struct {
	Id int64
	Lat float64  // In degrees
	Lon float64  // In degrees
	Attrs []Attribute
}

func NewMap() *Map {
	return &Map{
		make(map[int64]*Node),
		make(chan *osm.PrimitiveBlock),
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

func (m *Map) decodeDenseNodes(dn *osm.DenseNodes, bp *blockParams) ([]Node, error) {
	ids := dn.GetId()
	lats := dn.GetLat()
	lons := dn.GetLon()
	kvs := dn.GetKeysVals()
	if len(ids) != len(lats) || len(ids) != len(lons) {
		return nil, errors.New(fmt.Sprintf("Incorrect DB lengths: %d %d %d",
			len(ids), len(lats), len(lons)))
	}
	nodes := make([]Node, len(ids))

	// Initial delta-encoding last-values
	var lid int64
	var llat int64
	var llon int64
	kvi := 0
	for i, n := range nodes {
		lid += ids[i]
		llat += lats[i]
		llon += lons[i]
		n.Id = lid
		n.Lat = 1e-9 * float64(bp.latOffset + (bp.granularity * llat))
		n.Lon = 1e-9 * float64(bp.lonOffset + (bp.granularity * llon))
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

func (m *Map) decodeWay(w *osm.Way, bp *blockParams) error {
	return nil
}

func (m *Map) decodeRelation(w *osm.Relation, bp *blockParams) error {
	return nil
}

func (m *Map) decodeBlock(pb *osm.PrimitiveBlock) (*blockData, error) {
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
			nodes, err := m.decodeDenseNodes(dn, bparams)
			if err != nil {
				return nil, err
			}
			bdata.nodes = nodes
		}
		for _, w := range pg.GetWays() {
			if err := m.decodeWay(w, bparams); err != nil {
				return nil, err
			}
		}
		for _, r := range pg.GetRelations() {
			if err := m.decodeRelation(r, bparams); err != nil {
				return nil, err
			}
		}
	}
	return bdata, nil
}

func (m *Map) decodeBlockFunc() {
	for priblock := range m.blockCh {
		if priblock == nil {
			break
		}
		bd, err := m.decodeBlock(priblock)
		if err != nil {
			log.Print("Block decode failed!")  // @@@ TODO(jmacd)
			return
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
		for _, n := range bd.nodes {
			m.Nodes[n.Id] = &n
		}
	}
	m.doneCh <- true
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
		var blob osm.Blob
		if err = proto.Unmarshal(blobb, &blob); err != nil {
			return err
		}
		enc := "unknown"
		var data []byte

		// Uncompress the raw data, if necessary
		//
		// TODO(jmacd) Move this into a function. Do it
		// synchronously for the header, let a goproc
		// decompress for ordinary blocks.
		switch {
		case blob.Raw != nil:
			enc = "raw"
			data = blob.Raw
		case blob.ZlibData != nil:
			enc = "zlib"
			zr, err := zlib.NewReader(bytes.NewReader(blob.ZlibData))
			if err != nil {
				return err
			}
			defer zr.Close()
			if data, err = readFixed(zr, blob.GetRawSize()); err != nil {
				return err
			}
		case blob.LzmaData != nil:
			enc = "lzma"
		}
		if data == nil {
			return errors.New("Unsupported OSM data encoding: " + enc)
		}

		// Now process each blob
		// log.Printf("Read a blob %s type %s size %d / %d", 
		// 	bh.GetType(), enc, bsize, blob.GetRawSize())
		switch bh.GetType() {
		case "OSMHeader":
			var hdrblock osm.HeaderBlock
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
		case "OSMData":
			priblock := &osm.PrimitiveBlock{}
			if err := proto.Unmarshal(data, priblock); err != nil {
				return err
			}
			m.blockCh <- priblock
		default:
			return errors.New("Unknown OSM blob type: " + bh.GetType())
		}
	}
	for i := 0; i < numReaderProcs; i++ {
		m.blockCh <- nil
	}
	var _ = <- m.doneCh
	log.Println("Finished processing", nread, "bytes", len(m.Nodes), "nodes")
	return nil
}
