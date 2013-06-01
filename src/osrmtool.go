package main

import "database/sql"
import "flag"
import "hash/crc32"
import "log"

import "common"
import "data"
import "errors"
import "geo"

var osrmDir = flag.String("osrm_dir", "/home/jmacd/src/Project-OSRM",
	"A directory that contains osrm-routed and server.ini")

func main() {
	data.Main(programBody)
}

type LocId struct {
	Id int64
	geo.CityStateLoc
}

type OsrmTool struct {
	*data.ConvoyData
}

func (osrm *OsrmTool) fillTableFor(cslocs []*LocId) {
	for _, cslFrom := range cslocs {
		var dests []*LocId
		for _, cslTo := range cslocs {
			toName := cslTo.String()
			fromName := cslFrom.String()
			if toName == fromName {
				continue
			}
			tc := crc32.ChecksumIEEE([]byte(toName))
			fc := crc32.ChecksumIEEE([]byte(fromName))
			if tc == fc {
				log.Println("CRC32 hash collision", toName, fromName)
			}
			if tc <= fc {
				dests = append(dests, cslTo)
			}
		}

		log.Println(cslFrom, "has", len(dests), "destinations")
	}
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

	osrmTool.fillTableFor(cslocs)

	return nil
}