package main

import "database/sql"
import "flag"
import "fmt"
import "log"
import "code.google.com/p/go.net/html/atom"

import "data"
import "common"
import "scraper"

var dry_run = flag.Bool("dry_run", false, "")
var show_locations = flag.Bool("show_locations", false, "")
var show_corrections = flag.Bool("show_corrections", false, "")
var show_load_places = flag.Bool("show_load_places", false, "")
var try_finding = flag.String("try_finding", "", "")
var http_port = flag.Int("http_port", 8000, "")
var xvfb_port_offset = flag.Int("xvfb_port_offset", 1, "")

var try_spell_correction = true

type coordinates struct {
	lat, long float64  // In degrees
}

type CityFinder struct {
	missingStmt *sql.Stmt
	addCorStmt *sql.Stmt
	hasCorStmt *sql.Stmt
	addLocStmt *sql.Stmt
	hasLocStmt *sql.Stmt
	addGoogUnkStmt *sql.Stmt
	hasGoogUnkStmt *sql.Stmt
	addWikiUnkStmt *sql.Stmt
	hasWikiUnkStmt *sql.Stmt
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
	if cf.addCorStmt, err = data.InsertQuery(db, data.Corrections,
		"InCity", "InState", "OutCity", "OutState", "Determined"); err != nil {
		return nil, err
	}
	if cf.addLocStmt, err = data.InsertQuery(db, data.Locations,
		"LocCity", "LocState", "Latitude", "Longitude", "Determined"); err != nil {
		return nil, err
	}
	if cf.addGoogUnkStmt, err = data.InsertQuery(db, data.GoogleUnknown,
		"UnknownCity", "UnknownState"); err != nil {
		return nil, err
	}
	if cf.addWikiUnkStmt, err = data.InsertQuery(db, data.WikipediaUnknown,
		"UnknownUri"); err != nil {
		return nil, err
	}
	if cf.hasCorStmt, err = data.SelectQuery(db, data.Corrections, 
		"InCity", "InState"); err != nil {
		return nil, err
	}
	if cf.hasLocStmt, err = data.SelectQuery(db, data.Locations,
		"LocCity", "LocState"); err != nil {
		return nil, err
	}
	if cf.hasGoogUnkStmt, err = data.SelectQuery(db, data.GoogleUnknown,
		"UnknownCity", "UnknownState"); err != nil {
		return nil, err
	}
	if cf.hasWikiUnkStmt, err = data.SelectQuery(db, data.WikipediaUnknown,
		"UnknownUri"); err != nil {
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

// TODO(jmacd) Use data.ForAll
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

func (cf *CityFinder) hasLocation(cs common.CityState) (bool, error) {
	return data.HasRows(cf.hasLocStmt, cs.City, common.StateCode(cs.State))
}

func (cf *CityFinder) hasCorrection(cs common.CityState) (bool, error) {
	return data.HasRows(cf.hasCorStmt, cs.City, common.StateCode(cs.State))
}

func (cf *CityFinder) hasGoogleUnknown(cs common.CityState) (bool, error) {
	return data.HasRows(cf.hasGoogUnkStmt, cs.City, common.StateCode(cs.State))
}

func (cf *CityFinder) hasWikipediaUnknown(uri string) (bool, error) {
	return data.HasRows(cf.hasWikiUnkStmt, uri)
}

func (cf *CityFinder) getLocFromWiki(uri string) (coordinates, []byte, error) {
	var c coordinates
	xml, err := common.GetUrl(common.WikiHost, uri, "")
	if err != nil {
		return c, nil, err
	}
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
	return c, xml, nil
}

func (cf *CityFinder) tryLocFromWiki(urip *string, csp *common.CityState, spellDet *string) (coordinates, error) {
	var c coordinates
	hasUnk, err := cf.hasWikipediaUnknown(*urip)
	if err != nil {
		return c, err
	}
	if hasUnk {
		return c, nil
	}	
	c, xml, err := cf.getLocFromWiki(*urip)
	if err != nil {
		return c, err
	}
	if c.lat != 0 && c.long != 0 {
		return c, nil
	}
	ambiguous := false
	uris := []string{}
	err = scraper.ParseXml(xml, atom.A, "href", 
		func (value string) func (text string) {
 			if value == common.WikiDisambiguationUri {
				ambiguous = true
			}
			uris = append(uris, value)
			return nil
		})
	if ambiguous {
		//log.Println("Ambiguous - Uris", uris)
		for _, uri := range uris {
			cs, has := common.WikiUrlToCityState(uri)
			if has {
				c, _, err := cf.getLocFromWiki(uri)
				if err != nil {
					return c, err
				}
				if c.lat != 0 && c.long != 0 {
					*csp = cs
					*urip = uri
					*spellDet = "wiki-ambiguous"
					return c, nil
				}
			}
		}
	}
	_, err = cf.addWikiUnkStmt.Exec(*urip)
	if err != nil {
		return c, err
	}
	return c, nil
}
	
func (cf *CityFinder) tryFindingCoords(
	missing, spelling common.CityState, wikiUri, spellDet string) (bool, error) {
	hasLoc, err := cf.hasLocation(spelling)
	if err != nil {
		return false, err
	}

	var c coordinates
	if !hasLoc {
		c, err = cf.tryLocFromWiki(&wikiUri, &spelling, &spellDet)
		if err != nil {
			return false, err
		}
		if c.lat == 0 || c.long == 0 {
			log.Printf("%s: city not found (%s)",
				spelling, wikiUri)
			return false, nil
		}
		// "spelling" may have changed, updated hasLoc
		hasLoc, err = cf.hasLocation(spelling)
		if err != nil {
			return false, err
		}
	}
	spellingStateCode := common.StateCode(spelling.State)
	if missing.City != spelling.City || missing.State != spellingStateCode {
 		hasCor, err := cf.hasCorrection(missing)
		if err != nil {
			return false, err
		}
		if !hasCor {
			log.Printf("(%s, %s) -> (%s, %s) correction added (%s)", 
				missing.City, missing.State, 
				spelling.City, spellingStateCode, wikiUri)
			_, err := cf.addCorStmt.Exec(missing.City, missing.State, 
				spelling.City, spellingStateCode, spellDet)
			if err != nil {
				return false, err
			}
		}
	}
	if !hasLoc {
		log.Printf("(%s) coords %.3f,%.3f (%s)", spelling, c.lat, c.long, wikiUri)
		_, err = cf.addLocStmt.Exec(
			spelling.City, common.StateCode(spelling.State), c.lat, c.long, 
			wikiUri)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (cf *CityFinder) tryMissingCity(missing common.CityState) error {
	cities := common.GuessCityNames(missing)
	for _, city := range cities {
		found, err := cf.tryFindingCoords(missing, city, city.WikiUri(), "expanded")
		if err != nil {
			return err
		}
		if found {
			return nil
		}
	}
	if !try_spell_correction {
		return nil
	}
	for _, city := range cities {
		hasUnk, err := cf.hasGoogleUnknown(city)
		if err != nil {
			return err
		}
		if hasUnk {
			continue
		}
		spelling, wikiUri, spellDet, err := common.CorrectCitySpelling(city)
		if err != nil {
			// Typically this means daily search quota exceeded.
			log.Println("Spell correction failed -- disabling")
			try_spell_correction = false
			return nil
		}
		found, err := cf.tryFindingCoords(
			missing, spelling, wikiUri, spellDet)
		if err != nil {
			return err
		}
		if found {
			return nil
		}
		_, err = cf.addGoogUnkStmt.Exec(
			city.City, common.StateCode(city.State))
		if err != nil {
			return err
		}
	}
	return nil
}

func (cf *CityFinder) findMissingCities() error {
	count := 0
	ret := doAll(cf.missingStmt, func (cs common.CityState) error {
		count++
		if *dry_run {
			log.Println("Missing", cs)
			return nil
		}
		err := cf.tryMissingCity(cs)
		if err != nil {
			log.Printf("Error on %s: %s", cs, err)
		}
		return nil
	})
	log.Println("Queried", count, "cities")
	return ret
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
