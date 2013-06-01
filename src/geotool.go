package main

import "database/sql"
import "flag"
import "fmt"
import "log"
import "code.google.com/p/go.net/html/atom"

import "data"
import "common"
import "geo"
import "scraper"

var show_locations = flag.Bool("show_locations", false, "")
var show_corrections = flag.Bool("show_corrections", false, "")
var show_load_places = flag.Bool("show_load_places", false, "")
var show_load_pairs = flag.Bool("show_load_pairs", false, "")
var show_missing_cities = flag.Bool("show_missing_cities", false, "")
var try_finding = flag.String("try_finding", "", "")

var try_spell_correction = true

type CityFinder struct {
	data.ConvoyData
}

func (cf *CityFinder) getLocFromWiki(uri string) (geo.SphereCoords, []byte, error) {
	var c geo.SphereCoords
	xml, err := common.GetUrl(common.WikiHost, uri, "")
	if err != nil {
		return c, nil, err
	}
	err = scraper.ParseXml(xml, atom.Span, "class",
		func(value string) func(text string) {
			switch value {
			case "latitude":
				return func(text string) {
					c.Lat = geo.StringToDegrees(text)
				}
			case "longitude":
				return func(text string) {
					c.Long = geo.StringToDegrees(text)
				}
			}
			return nil
		})
	return c, xml, nil
}

func (cf *CityFinder) tryLocFromWiki(urip *string, csp *common.CityState, spellDet *string) (geo.SphereCoords, error) {
	var c geo.SphereCoords
	hasUnk, err := cf.HasWikipediaUnknown(*urip)
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
	if c.Defined() {
		return c, nil
	}
	ambiguous := false
	uris := []string{}
	err = scraper.ParseXml(xml, atom.A, "href",
		func(value string) func(text string) {
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
				if c.Defined() {
					*csp = cs
					*urip = uri
					*spellDet = "wiki-ambiguous"
					return c, nil
				}
			}
		}
	}
	err = cf.AddWikipediaUnknown(*urip)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (cf *CityFinder) tryFindingCoords(
	missing, spelling common.CityState, wikiUri, spellDet string) (bool, error) {
	hasLoc, err := cf.HasLocation(spelling)
	if err != nil {
		return false, err
	}

	var c geo.SphereCoords
	if !hasLoc {
		c, err = cf.tryLocFromWiki(&wikiUri, &spelling, &spellDet)
		if err != nil {
			return false, err
		}
		if !c.Defined() {
			log.Printf("(%s) city not found (%s)",
				spelling, wikiUri)
			return false, nil
		}
		// "spelling" may have changed, updated hasLoc
		hasLoc, err = cf.HasLocation(spelling)
		if err != nil {
			return false, err
		}
	}
	spelling.State = common.StateCode(spelling.State)
	if !missing.Equals(spelling) {
		hasCor, err := cf.HasCorrection(missing)
		if err != nil {
			return false, err
		}
		if !hasCor {
			log.Printf("(%s) -> (%s) correction added (%s)",
				missing, spelling, wikiUri)
			err = cf.AddCorrection(missing, spelling, spellDet)
			if err != nil {
				return false, err
			}
		}
	}
	if !hasLoc {
		log.Printf("(%s) coords %v (%s)", spelling, c, wikiUri)
		if err := cf.AddLocation(spelling, c, wikiUri); err != nil {
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
		hasUnk, err := cf.HasGoogleUnknown(city)
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
		city.State = common.StateCode(city.State)
		err = cf.AddGoogleUnknown(city)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cf *CityFinder) findMissingCities() error {
	count := 0
	ret := cf.ForAllMissingCities(func(cs common.CityState) error {
		count++
		err := cf.tryMissingCity(cs)
		if err != nil {
			log.Printf("Error on %s: %s", cs, err)
		}
		return nil
	})
	log.Println("Queried", count, "cities")
	return ret
}

func NewCityFinder(db *sql.DB) (*CityFinder, error) {
	cd, err := data.NewConvoyData(db)
	if err != nil {
		return nil, err
	}
	return &CityFinder{*cd}, nil
}

func main() {
	data.Main(programBody)
}

func programBody(db *sql.DB) error {
	flag.Parse()

	cf, err := NewCityFinder(db)
	if err != nil {
		return err
	}

	switch {
	case *show_locations:
		cf.ForAllLocations(func (id int64, csl geo.CityStateLoc) error {
			fmt.Println("[", id, "] ", csl.CityState, "->", csl.SphereCoords)
			return nil
		})
	case *show_corrections:
		cf.ForAllCorrections(func (from, to common.CityState) error {
			fmt.Println(from, "->", to)
			return nil
		})
	case *show_load_places:
		cf.ForAllLoadPlaces(func (cs common.CityState) error {
			fmt.Println(cs)
			return nil
		})
	case *show_missing_cities:
		cf.ForAllMissingCities(func (cs common.CityState) error {
			fmt.Println(cs)
			return nil
		})
	case *show_load_pairs:
		samefunc := func (from, to geo.CityStateLoc) error {
			fmt.Printf("%v -> %v\n", from, to)
			return nil
		}
		cf.ForAllLoadPairs(samefunc, samefunc)
	case len(*try_finding) != 0:
		cs := common.ParseCityState(*try_finding)
		if err = cf.tryMissingCity(cs); err != nil {
			return err
		}
	default:
		if err = cf.findMissingCities(); err != nil {
			return err
		}
	}
	return nil
}
