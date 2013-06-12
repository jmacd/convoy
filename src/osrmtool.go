package main

import "bufio"
import "bytes"
import "compress/zlib"
import "database/sql"
import "encoding/json"
import "encoding/binary"
import "errors"
import "flag"
import "fmt"
import "hash/crc32"
import "io"
import "io/ioutil"
import "log"
import "path"
import "os"
import "sort"
import "time"

import "common"
import "data"
import "geo"

var osrmDir = flag.String("osrm_dir", "/home/jmacd/src/Project-OSRM",
	"A directory that contains osrm-routed and server.ini")
var osrmHost = flag.String("osrm_host", "localhost:5000",
	"Default host:port for OSRM")
var distanceDir = flag.String("distance_dir", "/home/jmacd/OSM/Distance",
	"A directory for writing distance-table files")

func main() {
	data.Main(programBody)
}

type IdPair struct {
	From, To int64
}

type LocId struct {
	Id int64
	geo.CityStateLoc
}

type PairStat struct {
	meters, seconds int32
}

type LocPair struct {
	from, to *LocId
	ch chan<- *LocPair
	PairStat
	err error
}

type PairMap map[IdPair]PairStat

type OsrmTool struct {
	*data.ConvoyData
}

type OsrmRoute struct {
	// Version float64
	Status int `json:"status"`
	StatusMessage string `json:"status_message"`
	// RouteGeometry string
	// RouteInstructions string
	RouteSummary *OsrmSummary `json:"route_summary"`
}

type OsrmSummary struct {
	TotalDistance int32 `json:"total_distance"`
	TotalTime int32 `json:"total_time"`
	// StartPoint string
	// EndPoint string
}

type Destinations []*LocPair

type DestSort struct {
	Destinations
}

type NoRouteError struct {}

func (_ NoRouteError) Error() string {
	return "Cannot find route between points"
}

func (ds DestSort) Less(i, j int) bool {
	return ds.Destinations[i].to.Id < ds.Destinations[j].to.Id
}
func (ds DestSort) Len() int {
	return len(ds.Destinations)
}
func (ds DestSort) Swap(i, j int) {
	ds.Destinations[i], ds.Destinations[j] = ds.Destinations[j], ds.Destinations[i] 
}

func dtFile(c *LocId) string {
	return path.Join(*distanceDir, c.CityState.String())
}

func isDestinationFrom(from, to string) bool {
	if from == to {
		return false
	}
	var c string
	if to < from {
		c = to + from
	} else {
		c = from + to
	}
	return crc32.ChecksumIEEE([]byte(c)) & 0x1000 != 0
}

func (osrm *OsrmTool) fillTableFor(cslocs []*LocId, ch chan<- *LocPair) {
	for _, cslFrom := range cslocs {
		var dests []*LocId
		for _, cslTo := range cslocs {
			if isDestinationFrom(cslFrom.String(), cslTo.String()) {
				dests = append(dests, cslTo)
			}
		}
		err := osrm.fillDistanceTable(cslFrom, dests, ch)
		if err != nil {
			log.Println("Distance table:", cslFrom, err)
		}
	}
}

func (osrm *OsrmTool) readDistanceTable(from *LocId) (PairMap, error) {
	m := make(PairMap)
	p := dtFile(from)
	table, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	b := bytes.NewBuffer(table)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	rb := bufio.NewReader(r)
	for {
		i1, err1 := binary.ReadVarint(rb)
		if err1 == io.EOF {
			return m, nil
		}
		if err1 != nil {
			return nil, err1
		}
		i2, err2 := binary.ReadVarint(rb)
		if err2 != nil {
			return nil, err2
		}
		i3, err3 := binary.ReadVarint(rb)
		if err3 != nil {
			return nil, err3
		}
		m[IdPair{from.Id,i1}] = PairStat{int32(i2), int32(i3)}
	}
}

func (osrm *OsrmTool) fillDistanceTable(from *LocId, dests []*LocId, ch chan<- *LocPair) error {
	existPairs, err := osrm.readDistanceTable(from)
	if err != nil {
		existPairs = nil
	}
	log.Println(from, "has", len(dests), "destinations, read", len(existPairs))
	rch := make(chan *LocPair, len(dests))
	var lps []*LocPair
	waits := 0
	for _, d := range dests {
		lp := &LocPair{from: from, to: d, ch: rch}
		lps = append(lps, lp)
		if exist, has := existPairs[IdPair{from.Id, d.Id}]; has {
			lp.PairStat = exist
		} else {
			ch <- lp
			waits++
		}
	}
	for i := 0; i < waits; i++ { 
		<- rch
	}
	sort.Sort(DestSort{lps})
	noroute := 0
	
	msbuf := make([]byte, 3 * 5 * len(dests))  // 5 = max int32 -> varint
	mspos := 0
	for _, lp := range lps {
		if lp.err != nil {
			if _, ok := lp.err.(*NoRouteError); ok {
				noroute++
			} else {
				log.Printf("[%v] %s error %v", lp.to.Id, lp.to.CityStateLoc, lp.err)
			}
			mspos += binary.PutVarint(msbuf[mspos:mspos+5], 0)
			mspos += binary.PutVarint(msbuf[mspos:mspos+5], 0)
			mspos += binary.PutVarint(msbuf[mspos:mspos+5], 0)
		} else {
			mspos += binary.PutVarint(msbuf[mspos:mspos+5], int64(lp.to.Id))
			mspos += binary.PutVarint(msbuf[mspos:mspos+5], int64(lp.meters))
			mspos += binary.PutVarint(msbuf[mspos:mspos+5], int64(lp.seconds))
		}
	}
	if noroute != 0 {
		log.Printf("%d no-route errors", noroute)
	}
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(msbuf[0:mspos])
	w.Close()
	p := dtFile(from)
	return ioutil.WriteFile(p, b.Bytes(), os.ModePerm)
}		

func (osrm *OsrmTool) Viaroute(lp *LocPair) ([]byte, error) {
	url := "/viaroute"
	query := fmt.Sprintf("?loc=%.4f,%.4f&loc=%.4f,%.4f", 
		lp.from.SphereCoords.Lat, lp.from.SphereCoords.Long, 
		lp.to.SphereCoords.Lat, lp.to.SphereCoords.Long)
	return common.GetUrlFast(*osrmHost, url, query)
}

func (osrm *OsrmTool) computeDistance(lp *LocPair) error {
	data, err := osrm.Viaroute(lp)
	if err != nil {
		return err
	}
	var route OsrmRoute 

	if err := json.Unmarshal(data, &route); err != nil {
		return err
	}
	if route.Status == 207 {
		return &NoRouteError{}
	} else if route.Status != 0 {
		return errors.New(route.StatusMessage)
	} else if route.RouteSummary == nil {
		return errors.New("No route summary")
	}
	lp.meters = route.RouteSummary.TotalDistance
	lp.seconds = route.RouteSummary.TotalTime
	// log.Printf("%v -> %v: %.0f m %.0f s", 
	// 	lp.from,
	// 	lp.to,
	// 	lp.meters,
	// 	lp.seconds)
	return nil
}

func (osrm *OsrmTool) computeDistanceAndReturn(lp *LocPair) {
	lp.err = osrm.computeDistance(lp)
	lp.ch <- lp
}

func programBody(db *sql.DB) error {
	routed := *osrmDir + "/osrm-routed"
	servini := *osrmDir + "/server.ini"
	osrm, err := common.StartProcess(routed, []string{"NOENV=yes"}, servini)
	if err != nil {
		return err
	}
	defer osrm.Cleanup()
	
	cd, err := data.NewConvoyData(db)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 30)
	
	osrmTool := &OsrmTool{cd}
	var cslocs []*LocId
	if err = osrmTool.ForAllLocations(func (id int64, csl geo.CityStateLoc) error {
		cslocs = append(cslocs, &LocId{id, csl})
		return nil
	}); err != nil {
		return err
	}

	ch := make(chan *LocPair, common.NumCPU())
	d0 := make(chan bool)
	for i := 0; i < common.NumCPU(); i++ {
		go func() {
			for lp := range ch {
				osrmTool.computeDistanceAndReturn(lp)
			}
			d0 <- true
		}()
	}

	osrmTool.fillTableFor(cslocs, ch)

	close(ch)

	for i := 0; i < common.NumCPU(); i++ {
		<- d0
	}
	
	return nil
}
