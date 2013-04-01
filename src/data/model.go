package data

import "database/sql"
import "log"

import "common"
import "geo"

type ConvoyData struct {
	getAllMissingPlaces    *sql.Stmt
	addCorrection          *sql.Stmt
	hasCorrection          *sql.Stmt
	addLocation            *sql.Stmt
	hasLocation            *sql.Stmt
	addGoogleUnknown       *sql.Stmt
	hasGoogleUnkown        *sql.Stmt
	addWikiUnknown         *sql.Stmt
	hasWikiUnknown         *sql.Stmt
	getAllLocationPlaces   *sql.Stmt
	getAllCorrectionPlaces *sql.Stmt
	getAllLoadPlaces       *sql.Stmt
	getAllLoadPlacePairs   *sql.Stmt
	getAllCorrections      *sql.Stmt
	getAllLocations        *sql.Stmt
}

const (
	Corrections       TableName = "Corrections"
	Locations         TableName = "Locations"
	TruckLoads        TableName = "TruckLoads"
	LoadCityStates    TableName = "LoadCityStates"
	GeoCityStates     TableName = "GeoCityStates"
	GoogleUnknown     TableName = "GoogleUnknown"
	WikipediaUnknown  TableName = "WikipediaUnknown"
	UnknownCityStates TableName = "UnknownCityStates"
)

func NewConvoyData(db *sql.DB) (*ConvoyData, error) {
	var err error
	cd := &ConvoyData{}
	if cd.addCorrection, err = InsertQuery(db, Corrections,
		"InCity", "InState", "OutCity", "OutState", "Determined"); err != nil {
		return nil, err
	}
	if cd.addLocation, err = InsertQuery(db, Locations,
		"LocCity", "LocState", "Latitude", "Longitude", "Determined"); err != nil {
		return nil, err
	}
	if cd.addGoogleUnknown, err = InsertQuery(db, GoogleUnknown,
		"UnknownCity", "UnknownState"); err != nil {
		return nil, err
	}
	if cd.addWikiUnknown, err = InsertQuery(db, WikipediaUnknown,
		"UnknownUri"); err != nil {
		return nil, err
	}
	if cd.hasCorrection, err = SelectWhereQuery(db, Corrections,
		"InCity", "InState"); err != nil {
		return nil, err
	}
	if cd.hasLocation, err = SelectWhereQuery(db, Locations,
		"LocCity", "LocState"); err != nil {
		return nil, err
	}
	if cd.hasGoogleUnkown, err = SelectWhereQuery(db, GoogleUnknown,
		"UnknownCity", "UnknownState"); err != nil {
		return nil, err
	}
	if cd.hasWikiUnknown, err = SelectWhereQuery(db, WikipediaUnknown,
		"UnknownUri"); err != nil {
		return nil, err
	}
	if cd.getAllMissingPlaces, err = SelectGroupQuery(db, UnknownCityStates,
		"C", "S"); err != nil {
		return nil, err
	}
	if cd.getAllLocationPlaces, err = SelectGroupQuery(db, Locations,
		"LocCity", "LocState"); err != nil {
		return nil, err
	}
	if cd.getAllCorrectionPlaces, err = SelectGroupQuery(db, Corrections,
		"InCity", "InState"); err != nil {
		return nil, err
	}
	if cd.getAllLoadPlaces, err = SelectGroupQuery(db, LoadCityStates,
		"C", "S"); err != nil {
		return nil, err
	}
	if cd.getAllLoadPlacePairs, err = SelectGroupQuery(db, TruckLoads,
		"OriginCity", "OriginState", "DestCity", "DestState"); err != nil {
		return nil, err
	}
	if cd.getAllCorrections, err = SelectGroupQuery(db, Corrections,
		"InCity", "InState", "OutCity", "OutState"); err != nil {
		return nil, err
	}
	if cd.getAllLocations, err = SelectGroupQuery(db, Locations,
		"LocCity", "LocState", "Latitude", "Longitude"); err != nil {
		return nil, err
	}
	return cd, nil
}

func (cd *ConvoyData) HasLocation(cs common.CityState) (bool, error) {
	return HasRows(cd.hasLocation, cs.City, common.StateCode(cs.State))
}

func (cd *ConvoyData) HasCorrection(cs common.CityState) (bool, error) {
	return HasRows(cd.hasCorrection, cs.City, common.StateCode(cs.State))
}

func (cd *ConvoyData) HasGoogleUnknown(cs common.CityState) (bool, error) {
	return HasRows(cd.hasGoogleUnkown, cs.City, common.StateCode(cs.State))
}

func (cd *ConvoyData) HasWikipediaUnknown(uri string) (bool, error) {
	return HasRows(cd.hasWikiUnknown, uri)
}

func (cd *ConvoyData) AddWikipediaUnknown(uri string) error {
	_, err := cd.addWikiUnknown.Exec(uri)
	return err
}

func (cd *ConvoyData) AddGoogleUnknown(cs common.CityState) error {
	if cs.State != common.StateCode(cs.State) {
		panic("StateCode() not applied")
	}
	_, err := cd.addGoogleUnknown.Exec(cs.City, cs.State)
	return err
}

func (cd *ConvoyData) AddCorrection(from common.CityState,
	to common.CityState, det string) error {

	if from.State != common.StateCode(from.State) ||
		to.State != common.StateCode(to.State) {
		panic("StateCode() not applied")
	}
	_, err := cd.addCorrection.Exec(from.City, from.State, to.City, to.State, det)
	return err
}

func (cd *ConvoyData) AddLocation(cs common.CityState,
	loc geo.SphereCoords, uri string) error {
	if cs.State != common.StateCode(cs.State) {
		panic("StateCode() not applied")
	}
	_, err := cd.addLocation.Exec(cs.City, cs.State, loc.Lat, loc.Long, uri)
	if err != nil {
		return err
	}
	return err
}

func (cd *ConvoyData) ForAllLoadPlaces(csfunc func(common.CityState) error) error {
	return forAllCities(cd.getAllLoadPlaces, csfunc)
}

func (cd *ConvoyData) ForAllMissingCities(csfunc func(common.CityState) error) error {
	return forAllCities(cd.getAllMissingPlaces, csfunc)
}

func (cd *ConvoyData) ForAllLocations(lfunc func (common.CityState, geo.SphereCoords) error) error {
	var locCity, locState []byte
	var lat, long float64
	return ForAll(cd.getAllLocations, func () error {
		return lfunc(
			common.CityState{string(locCity), string(locState)},
			geo.SphereCoords{lat, long})
	}, &locCity, &locState, &lat, &long)
}

func (cd *ConvoyData) ForAllCorrections(
	cfunc func (from, to common.CityState) error) error {
	var fromCity, fromState, toCity, toState []byte
	return ForAll(cd.getAllCorrections, func () error {
		return cfunc(common.CityState{
			string(fromCity), string(fromState)},
			common.CityState{string(toCity),
			string(toState)})
	}, &fromCity, &fromState, &toCity, &toState)
}

func (cd *ConvoyData) ForAllLoadPairs(
	lfunc func(from, to common.CityState, 
		fromLoc, toLoc geo.SphereCoords) error) error {
	corrections := make(map[string]common.CityState)
	locations := make(map[string]geo.SphereCoords)
	if err := cd.ForAllCorrections(func (in, out common.CityState) error {
		corrections[in.String()] = out
		return nil
	}); err != nil {
		return err
	}
	if err := cd.ForAllLocations(
		func (loc common.CityState, spc geo.SphereCoords) error {
		locations[loc.String()] = spc
		return nil
	}); err != nil {
		return err
	}
	unresolved := 0
	output := make(map[string]bool)
	var fromCity, fromState, toCity, toState []byte
	if err := ForAll(cd.getAllLoadPlacePairs, func () error {
		from := common.CityState{string(fromCity), string(fromState)}
		to := common.CityState{string(toCity), string(toState)}
		if cs, has := corrections[from.String()]; has {
			from = cs
		}
		if cs, has := corrections[to.String()]; has {
			to = cs
		}
		fl, hasFl := locations[from.String()]
		tl, hasTl := locations[to.String()]
		if !hasFl || !hasTl {
			unresolved++
			return nil
		}
		if to.String() < from.String() {
			from, to = to, from
			fl, tl = tl, fl
		}

		comb := from.String() + "/" + to.String()
		if _, has := output[comb]; has {
			return nil
		}
		output[comb] = true
		return lfunc(from, to, fl, tl)
	}, &fromCity, &fromState, &toCity, &toState); err != nil {
		return err
	}
	log.Println("Skipped", unresolved, "city pairs")
	return nil
}

func forAllCities(
	stmt *sql.Stmt, csfunc func(common.CityState) error) error {
	var city, state []byte
	return ForAll(stmt, func() error {
		return csfunc(common.CityState{string(city),
			string(state)})
	}, &city, &state)
}
