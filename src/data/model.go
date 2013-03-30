package data

import "database/sql"
import "fmt"
import "log"

import "common"
import "geo"

type ConvoyData struct {
	missingStmt    *sql.Stmt
	addCorStmt     *sql.Stmt
	hasCorStmt     *sql.Stmt
	addLocStmt     *sql.Stmt
	hasLocStmt     *sql.Stmt
	addGoogUnkStmt *sql.Stmt
	hasGoogUnkStmt *sql.Stmt
	addWikiUnkStmt *sql.Stmt
	hasWikiUnkStmt *sql.Stmt
	getAllLocsStmt *sql.Stmt
	getAllCorrStmt *sql.Stmt
	getAllLoadStmt *sql.Stmt
}

func NewConvoyData(db *sql.DB) (*ConvoyData, error) {
	var err error
	cd := &ConvoyData{}
	// TODO(jmacd) Apparently this is slow because MySQL decides
	// to do sequential scan for very large "IN" expressions; fix.
	if cd.missingStmt, err = db.Prepare("SELECT C, S FROM (SELECT C, S FROM " + Table(LoadCityStates) + " GROUP BY C, S) AS Loads WHERE (C, S) NOT IN (SELECT C, S FROM " + Table(GeoCityStates) + " AS Places GROUP BY C, S)"); err != nil {
		return nil, err
	}
	if cd.addCorStmt, err = InsertQuery(db, Corrections,
		"InCity", "InState", "OutCity", "OutState", "Determined"); err != nil {
		return nil, err
	}
	if cd.addLocStmt, err = InsertQuery(db, Locations,
		"LocCity", "LocState", "Latitude", "Longitude", "Determined"); err != nil {
		return nil, err
	}
	if cd.addGoogUnkStmt, err = InsertQuery(db, GoogleUnknown,
		"UnknownCity", "UnknownState"); err != nil {
		return nil, err
	}
	if cd.addWikiUnkStmt, err = InsertQuery(db, WikipediaUnknown,
		"UnknownUri"); err != nil {
		return nil, err
	}
	if cd.hasCorStmt, err = SelectQuery(db, Corrections,
		"InCity", "InState"); err != nil {
		return nil, err
	}
	if cd.hasLocStmt, err = SelectQuery(db, Locations,
		"LocCity", "LocState"); err != nil {
		return nil, err
	}
	if cd.hasGoogUnkStmt, err = SelectQuery(db, GoogleUnknown,
		"UnknownCity", "UnknownState"); err != nil {
		return nil, err
	}
	if cd.hasWikiUnkStmt, err = SelectQuery(db, WikipediaUnknown,
		"UnknownUri"); err != nil {
		return nil, err
	}
	if cd.getAllLocsStmt, err = SelectQuery(db, Locations,
		"LocCity", "LocState"); err != nil {
		return nil, err
	}
	if cd.getAllCorrStmt, err = db.Prepare(
		"SELECT InCity, InState FROM " +
			Table(Corrections) +
			" GROUP BY InCity, InState"); err != nil {
		return nil, err
	}
	if cd.getAllLoadStmt, err = db.Prepare(
		"SELECT C, S FROM " +
			Table(LoadCityStates) +
			" GROUP BY C, S"); err != nil {
		return nil, err
	}
	return cd, nil
}

func (cd *ConvoyData) HasLocation(cs common.CityState) (bool, error) {
	return HasRows(cd.hasLocStmt, cs.City, common.StateCode(cs.State))
}

func (cd *ConvoyData) HasCorrection(cs common.CityState) (bool, error) {
	return HasRows(cd.hasCorStmt, cs.City, common.StateCode(cs.State))
}

func (cd *ConvoyData) HasGoogleUnknown(cs common.CityState) (bool, error) {
	return HasRows(cd.hasGoogUnkStmt, cs.City, common.StateCode(cs.State))
}

func (cd *ConvoyData) HasWikipediaUnknown(uri string) (bool, error) {
	return HasRows(cd.hasWikiUnkStmt, uri)
}

func (cd *ConvoyData) AddWikipediaUnknown(uri string) error {
	_, err := cd.addWikiUnkStmt.Exec(uri)
	return err
}

func (cd *ConvoyData) AddGoogleUnknown(cs common.CityState) error {
	if cs.State != common.StateCode(cs.State) {
		panic("StateCode() not applied")
	}
	_, err := cd.addGoogUnkStmt.Exec(cs.City, cs.State)
	return err
}

func (cd *ConvoyData) AddCorrection(from common.CityState,
	to common.CityState, det string) error {

	if from.State != common.StateCode(from.State) ||
		to.State != common.StateCode(to.State) {
		panic("StateCode() not applied")
	}
	_, err := cd.addCorStmt.Exec(from.City, from.State, to.City, to.State, det)
	return err
}

func (cd *ConvoyData) AddLocation(cs common.CityState,
	loc geo.SphereCoords, uri string) error {
	if cs.State != common.StateCode(cs.State) {
		panic("StateCode() not applied")
	}
	_, err := cd.addLocStmt.Exec(cs.City, cs.State, loc.Lat, loc.Long, uri)
	if err != nil {
		return err
	}
	return err
}

func (cd *ConvoyData) ForAllMissingCities(csfunc func(common.CityState) error) error {
	return ForAllCities(cd.missingStmt, csfunc)
}

func (cd *ConvoyData) ShowAllLocations() {
	ShowAll(cd.getAllLocsStmt)
}

func (cd *ConvoyData) ShowAllCorrections() {
	ShowAll(cd.getAllCorrStmt)
}

func (cd *ConvoyData) ShowAllLoads() {
	ShowAll(cd.getAllLoadStmt)
}

func ForAllCities(
	stmt *sql.Stmt, csfunc func(common.CityState) error) error {
	var city, state []byte
	return ForAll(stmt, func() error {
		return csfunc(common.CityState{string(city),
			string(state)})
	}, &city, &state)
}

func ShowAll(stmt *sql.Stmt) {
	err := ForAllCities(stmt, func(cs common.CityState) error {
		fmt.Println(cs)
		return nil
	})
	if err != nil {
		log.Println("Error in ShowAll", err)
	}
}

// TODO(jmacd) Move this...
// func printCityDistances(db *sql.DB, tree *geo.Tree) error {
// 	stmt, err := db.Prepare(
// 		"SELECT LocCity, LocState, Latitude, Longitude FROM " +
// 		Table(Locations))
// 	if err != nil {
// 		return err
// 	}
// 	count := 0
// 	var city, state []byte
// 	var lat, long float64
// 	if err := ForAll(stmt, func () {
// 		count++
// 		var coords [3]geo.EarthLoc
// 		geo.LatLongDegreesToCoords(lat, long, coords[:])
// 		near := tree.FindNearest(coords[:])
// 		dist := geo.GreatCircleDistance(near.Point(), coords[:])
// 		log.Printf("%v, %v @ %.2f,%.2f nearest %.2fkm",
// 			string(city), string(state), lat, long, dist / 1000.0)
// 	}, &city, &state, &lat, &long); err != nil {
// 		return err
// 	}
// 	log.Println("Scanned", count, "cities")
// 	return nil
// }
