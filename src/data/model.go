package data

import "database/sql"
import "time"

import "boards"
import "common"
import "geo"
import "scraper"

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
	addRoadDistance        *sql.Stmt
	hasRoadDistance        *sql.Stmt
	getAllLocationPlaces   *sql.Stmt
	getAllCorrectionPlaces *sql.Stmt
	getAllLoadPlaces       *sql.Stmt
	getAllLoadPlacePairs   *sql.Stmt
	getAllCorrections      *sql.Stmt
	getAllLocations        *sql.Stmt
	getAllLoads            *sql.Stmt
	getAllScrapes          *sql.Stmt
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
	RoadDistance      TableName = "RoadDistance"
	Scrapes           TableName = "Scrapes"
)

type CityFunc func(common.CityState) error
type CityLocFunc func(geo.CityStateLoc) error
type CityPairFunc func(from, to common.CityState) error
type CityPairLocFunc func(from, to geo.CityStateLoc) error
type LoadFunc func(load boards.Load) error
type ScrapeFunc func(scrape scraper.Scrape) error

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
	if cd.addRoadDistance, err = InsertQuery(db, RoadDistance,
		"SourceCity", "SourceState",
		"DestCity", "DestState", "Kilometers"); err != nil {
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
	if cd.hasRoadDistance, err = SelectWhereQuery(db, RoadDistance,
		"SourceCity", "SourceState",
		"DestCity", "DestState"); err != nil {
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
		"OriginCity", "OriginState",
		"DestCity", "DestState"); err != nil {
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
	if cd.getAllLoads, err = SelectAllQuery(db, TruckLoads,
		"ScrapeId", "PickupDate", "OriginState", "OriginCity",
		"DestState", "DestCity", "LoadType", "Length", "Weight",
		"Equipment", "Price", "Stops", "Phone"); err != nil {
		return nil, err
	}
	if cd.getAllScrapes, err = SelectAllQuery(db, Scrapes,
		"ScrapeId", "StartTime", "FinishTime"); err != nil {
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

func (cd *ConvoyData) HasRoadDistance(src, dest common.CityState) (bool, error) {
	return HasRows(cd.hasRoadDistance, src.City, src.State, dest.City, dest.State)
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

func (cd *ConvoyData) AddRoadDistance(src common.CityState,
	dest common.CityState, kilometers int) error {

	if src.State != common.StateCode(src.State) ||
		dest.State != common.StateCode(dest.State) {
		panic("StateCode() not applied")
	}
	_, err := cd.addRoadDistance.Exec(src.City, src.State, dest.City, dest.State, kilometers)
	return err
}

func (cd *ConvoyData) ForAllLoadPlaces(csfunc CityFunc) error {
	return forAllCities(cd.getAllLoadPlaces, csfunc)
}

func (cd *ConvoyData) ForAllMissingCities(csfunc CityFunc) error {
	return forAllCities(cd.getAllMissingPlaces, csfunc)
}

func (cd *ConvoyData) ForAllLocations(lfunc CityLocFunc) error {
	var locCity, locState []byte
	var lat, long float64
	return ForAll(cd.getAllLocations, func() error {
		return lfunc(
			geo.CityStateLoc{common.CityState{string(locCity), string(locState)},
				geo.SphereCoords{lat, long}})
	}, &locCity, &locState, &lat, &long)
}

func (cd *ConvoyData) ForAllCorrections(cfunc CityPairFunc) error {
	var fromCity, fromState, toCity, toState []byte
	return ForAll(cd.getAllCorrections, func() error {
		return cfunc(common.CityState{
			string(fromCity), string(fromState)},
			common.CityState{string(toCity),
				string(toState)})
	}, &fromCity, &fromState, &toCity, &toState)
}

func (cd *ConvoyData) ForAllLoadPairsMissingDistance(mfunc CityPairLocFunc) error {
	ufunc := func(from, to geo.CityStateLoc) error {
		return nil
	}
	lfunc := func(from, to geo.CityStateLoc) error {
		has, err := cd.HasRoadDistance(from.CityState, to.CityState)
		if err != nil {
			return err
		}
		if has {
			return nil
		}
		return mfunc(from, to)
	}
	return cd.ForAllLoadPairs(lfunc, ufunc)
}

func (cd *ConvoyData) ForAllLoadPairs(loadFunc, undefFunc CityPairLocFunc) error {

	corrections := make(map[common.CityState]common.CityState)
	locations := make(map[common.CityState]geo.SphereCoords)
	if err := cd.ForAllCorrections(func(in, out common.CityState) error {
		corrections[in] = out
		return nil
	}); err != nil {
		return err
	}
	if err := cd.ForAllLocations(
		func(loc geo.CityStateLoc) error {
			locations[loc.CityState] = loc.SphereCoords
			return nil
		}); err != nil {
		return err
	}
	unresolved := 0
	output := make(map[string]bool)
	var fromCity, fromState, toCity, toState []byte
	if err := ForAll(cd.getAllLoadPlacePairs, func() error {
		from := common.CityState{string(fromCity), string(fromState)}
		to := common.CityState{string(toCity), string(toState)}
		if cs, has := corrections[from]; has {
			from = cs
		}
		if cs, has := corrections[to]; has {
			to = cs
		}
		fl, hasFl := locations[from]
		tl, hasTl := locations[to]
		if !hasFl || !hasTl {
			unresolved++
			return undefFunc(
				geo.CityStateLoc{CityState: from},
				geo.CityStateLoc{CityState: to})
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
		return loadFunc(geo.CityStateLoc{from, fl}, geo.CityStateLoc{to, tl})
	}, &fromCity, &fromState, &toCity, &toState); err != nil {
		return err
	}
	return nil
}

func forAllCities(stmt *sql.Stmt, csfunc CityFunc) error {
	var city, state []byte
	return ForAll(stmt, func() error {
		return csfunc(common.CityState{string(city),
			string(state)})
	}, &city, &state)
}

func (cd *ConvoyData) ForAllLoads(loadFunc LoadFunc) error {
	// The following is somewhat convoluted, avoids
	// "closure needs too many variables; runtime will reject it"
	var scrapeId int64
	var ints [4]int
	var strings [7][]byte
	var loadTime []byte
	return ForAll(cd.getAllLoads, func() error {
		tm, err := time.Parse(common.SqlDateFmt, string(loadTime))
		if err != nil {
			return err
		}
		return loadFunc(boards.Load{scrapeId, tm,
			common.CityState{string(strings[1]), string(strings[0])},
			common.CityState{string(strings[3]), string(strings[2])},
			string(strings[4]), ints[0], ints[1],
			string(strings[5]), ints[2], ints[3],
			string(strings[6])})
	}, &scrapeId, &loadTime, &strings[0], &strings[1], &strings[2], &strings[3],
		&strings[4], &ints[0], &ints[1], &strings[5],
		&ints[2], &ints[3], &strings[6])
}

func (cd *ConvoyData) ForAllScrapes(sfunc ScrapeFunc) error {
	var scrapeId int64
	var startTime, finishTime []byte
	return ForAll(cd.getAllScrapes, func () error {
		st, err := time.Parse(common.SqlDateFmt, string(startTime))
		if err != nil {
			return err
		}
		ft, err := time.Parse(common.SqlDateFmt, string(finishTime))
		if err != nil {
			return err
		}
		s := scraper.Scrape{scrapeId, st, ft}
		return sfunc(s)
	}, &scrapeId, &startTime, &finishTime)
}
