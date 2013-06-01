package main

import "database/sql"
import "encoding/json"
import "errors"
import "flag"
import "fmt"
import "hash/crc32"
import "log"
import "time"
import "runtime"

import "common"
import "data"
import "geo"

var osrmDir = flag.String("osrm_dir", "/home/jmacd/src/Project-OSRM",
	"A directory that contains osrm-routed and server.ini")
var osrmHost = flag.String("osrm_host", "localhost:5000",
	"Default host:port for OSRM")

func main() {
	data.Main(programBody)
}

type LocId struct {
	Id int64
	geo.CityStateLoc
}

type LocPair struct {
	from, to *LocId
}

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

func (osrm *OsrmTool) fillTableFor(cslocs []*LocId, ch chan<- LocPair) {
	for _, cslFrom := range cslocs {
		var dests []*LocId
		for _, cslTo := range cslocs {
			if isDestinationFrom(cslFrom.String(), cslTo.String()) {
				dests = append(dests, cslTo)
			}
		}
		osrm.fillDistanceTable(cslFrom, dests, ch)
	}
}

func (osrm *OsrmTool) fillDistanceTable(from *LocId, dests []*LocId, ch chan<- LocPair) {
	//log.Println(from, "has", len(dests), "destinations")
	for _, d := range dests {
		ch <- LocPair{from, d}
	}
}		

func (osrm *OsrmTool) Viaroute(lp LocPair) ([]byte, error) {
	url := "/viaroute"
	query := fmt.Sprintf("?loc=%.4f,%.4f&loc=%.4f,%.4f", 
		lp.from.SphereCoords.Lat, lp.from.SphereCoords.Long, 
		lp.to.SphereCoords.Lat, lp.to.SphereCoords.Long)
	return common.GetUrlFast(*osrmHost, url, query)
}

func (osrm *OsrmTool) computeDistance(lp LocPair) error {
	data, err := osrm.Viaroute(lp)
	if err != nil {
		return err
	}
	var route OsrmRoute 

	if err := json.Unmarshal(data, &route); err != nil {
		return err
	}
	if route.Status != 0 {
		return errors.New(route.StatusMessage)
	} else if route.RouteSummary == nil {
		return errors.New("No route summary")
	}

	//log.Printf("%s -> %s: %.2fkm %.2fhrs", from, d,
	//	(float64(route.RouteSummary.TotalDistance) / 1000.0),
	//	(float64(route.RouteSummary.TotalTime) / 3600.0))
	return nil
}

func programBody(db *sql.DB) error {
	routed := *osrmDir + "/osrm-routed"
	servini := *osrmDir + "/server.ini"
	osrm, err := common.StartProcess(routed, []string{"NOENV=yes"}, servini)
	if err != nil {
		return err
	}
	defer osrm.Cleanup()
	
	time.Sleep(time.Second * 30)
	cd, err := data.NewConvoyData(db)
	if err != nil {
		return err
	}
	
	osrmTool := &OsrmTool{cd}
	var cslocs []*LocId
	if err = osrmTool.ForAllLocations(func (id int64, csl geo.CityStateLoc) error {
		cslocs = append(cslocs, &LocId{id, csl})
		return nil
	}); err != nil {
		return err
	}

	if len(cslocs) <= 1 {
		return errors.New("Fewer than two locations!")
	}

	var success, failure int
	ch := make(chan LocPair, runtime.NumCPU())
	es := make(chan error, runtime.NumCPU())
	d0 := make(chan bool)
	d1 := make(chan bool)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for lp := range ch {
				es <- osrmTool.computeDistance(lp)
			}
			d0 <- true
		}()
	}
	go func() {
		for e := range es {
			if e == nil {
				success++
			} else {
				failure++
			}
		}
		d1 <- true
	}()

	osrmTool.fillTableFor(cslocs, ch)

	close(ch)

	for i := 0; i < runtime.NumCPU(); i++ {
		<- d0
	}
	
	close(es)
	<- d1
	log.Println("Success", success, "Failure", failure)
	return nil
}
