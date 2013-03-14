package main

import "database/sql"
import "flag"
import "fmt"
import "log"
import "code.google.com/p/go.net/html/atom"

import "data"
import "common"
import "scraper"

var show_locations = flag.Bool("show_locations", false, "")
var show_corrections = flag.Bool("show_corrections", false, "")
var show_load_places = flag.Bool("show_load_places", false, "")
var try_finding = flag.String("try_finding", "", "")
var http_port = flag.Int("http_port", 8000, "")
var xvfb_port_offset = flag.Int("xvfb_port_offset", 1, "")

type coordinates struct {
	lat, long float64  // In degrees
}

type CityFinder struct {
	missingStmt *sql.Stmt
	addCorStmt *sql.Stmt
	addLocStmt *sql.Stmt
	hasLocStmt *sql.Stmt
	hasCorStmt *sql.Stmt
	getAllLocsStmt *sql.Stmt
	getAllCorrStmt *sql.Stmt
	getAllLoadStmt *sql.Stmt
}

func NewCityFinder(db *sql.DB) (*CityFinder, error) {
	var err error
	cf := &CityFinder{}
	// TODO(jmacd) Understand why this query is so slow and figure out how to optimize it.
	if cf.missingStmt, err = db.Prepare("SELECT C, S FROM (SELECT C, S FROM " + data.Table(data.LoadCityStates) + " GROUP BY C, S) AS Loads WHERE (C, S) NOT IN (SELECT C, S FROM " + data.Table(data.GeoCityStates) + " AS Places GROUP BY C, S)"); err != nil {
		return nil, err
	}
	if cf.addCorStmt, err = db.Prepare("INSERT INTO " + 
		data.Table(data.Corrections) +
		" (InCity, InState, OutCity, OutState)" +
		" VALUES (?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if cf.addLocStmt, err = db.Prepare("INSERT INTO " +
		data.Table(data.Locations) +
		" (LocCity, LocState, Latitude, Longitude)" +
		" VALUES (?, ?, ?, ?)"); err != nil {
		return nil, err
	}
	if cf.hasLocStmt, err = db.Prepare("SELECT * FROM " +
		data.Table(data.Locations) +
		" WHERE LocCity = ? AND LocState = ?"); err != nil {
		return nil, err
	}
	if cf.hasCorStmt, err = db.Prepare("SELECT * FROM " +
		data.Table(data.Corrections) +
		" WHERE InCity = ? AND InState = ?"); err != nil {
		return nil, err
	}
	if cf.getAllLocsStmt, err = db.Prepare(
		"SELECT LocCity, LocState FROM " +
		data.Table(data.Locations) + 
		" GROUP BY LocCity, LocState"); err != nil {
		return nil, err
	}
	if cf.getAllCorrStmt, err = db.Prepare(
		"SELECT InCity, InState FROM " +
		data.Table(data.Corrections) +
		" GROUP BY InCity, InState"); err != nil {
		return nil, err
	}
	if cf.getAllLoadStmt, err = db.Prepare(
		"SELECT C, S FROM " +
		data.Table(data.LoadCityStates) + 
		" GROUP BY C, S"); err != nil {
		return nil, err
	}
	return cf, nil
}

func doAll(stmt *sql.Stmt, csfunc func (cs common.CityState) error) error {
	rows, err := stmt.Query()
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var city, state []byte
		if err := rows.Scan(&city, &state); err != nil {
			return err
		}
		if err := csfunc(common.CityState{string(city), string(state)}); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func hasRows(s *sql.Stmt, a ...interface{}) (bool, error) {
	has, err := s.Query(a...)
	if err != nil {
		return false, err
	}
	defer has.Close()
	if has.Next() {
		return true, nil
	}
	if err := has.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func (cf *CityFinder) hasLocation(cs common.CityState) (bool, error) {
	return hasRows(cf.hasLocStmt, cs.City, common.StateCode(cs.State))
}

func (cf *CityFinder) hasCorrection(cs common.CityState) (bool, error) {
	return hasRows(cf.hasCorStmt, cs.City, common.StateCode(cs.State))
}

func (cf *CityFinder) tryFindingCoords(missing common.CityState) error {
	// Missing comes directly from the board (is an abbreviation).
	name, uri, err := common.GuessWikiUri(missing)
	if err != nil {
		return err
	}
	hasLoc, err := cf.hasLocation(name)
	if err != nil {
		return err
	}
	nameStateCode := common.StateCode(name.State)

	if missing.City != name.City || missing.State != nameStateCode {
 		hasCor, err := cf.hasCorrection(missing)
		if err != nil {
			return err
		}
		if !hasCor {
			log.Printf("(%s, %s) -> (%s, %s) correction added (%s)", 
				missing.City, missing.State, name.City, nameStateCode, uri)
			_, err := cf.addCorStmt.Exec(missing.City, missing.State, 
				name.City, nameStateCode)
			if err != nil {
				return err
			}
		}
	}
	if hasLoc {
		return nil
	}

	xml, err := common.GetUrl(common.WikiHost, uri, "")
	if err != nil {
		return err
	}
	var c coordinates
	err = scraper.ParseXml(xml, atom.Span, "class", 
		func (value string) func (text string) {
		switch value {
		case "latitude":
			return func (text string) {
				c.lat = common.StringToDegrees(text)
			}
		case "longitude":
			return func (text string) {
				c.long = common.StringToDegrees(text)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if c.lat == 0 || c.long == 0 {
		log.Printf("(%s) -> (%s) city not found %s", missing, name, uri)
		return nil
	}
	log.Printf("(%s) coords %.3f,%.3f", name, c.lat, c.long)
	_, err = cf.addLocStmt.Exec(
		name.City, common.StateCode(name.State), c.lat, c.long)
	if err != nil {
		return err
	}

	return nil
}

func (cf *CityFinder) tryMissingCity(cs common.CityState) error {
	if err := cf.tryFindingCoords(cs); err != nil {
		log.Printf("Failed on %s: %s", cs, err)
	}
	// Keep going...
	return nil
}

func (cf *CityFinder) findMissingCities() error {
	return doAll(cf.missingStmt, func (cs common.CityState) error {
		return cf.tryMissingCity(cs)
	})
}

func showAll(stmt *sql.Stmt) error {
	return doAll(stmt, func (cs common.CityState) error {
		fmt.Println(cs)
		return nil
	})
}

func main() {
	flag.Parse()
	db, err := data.OpenDb()
	if err != nil {
		log.Fatal("Could not open database", err)
	}
	defer db.Close()

	cf, err := NewCityFinder(db)
	if err != nil {
		log.Fatal("NewCityFinder failed", err)		
	}

	switch {
	case *show_locations:
		showAll(cf.getAllLocsStmt)
	case *show_corrections:
		showAll(cf.getAllCorrStmt)
	case *show_load_places:
		showAll(cf.getAllLoadStmt)
	case len(*try_finding) != 0:
		cs := common.ParseCityState(*try_finding)
		if err = cf.tryMissingCity(cs); err != nil {
			log.Fatalf("Failed finding %s: %s", cs, err)
		}
	default:
		if err = cf.findMissingCities(); err != nil {
			log.Fatal("findMissingCities failed", err)
		}
	}
}
