package common

import "regexp"
import "strings"

const (
	WikiHost = "en.wikipedia.org"
	wikiBaseUri = "/wiki/"
)

var (
	wikiUrlRe = regexp.MustCompile(
		`^http://` + WikiHost + wikiBaseUri + `([^#]+)`)
	wikiCityStateRe = regexp.MustCompile(`(.*), ([^,]+)`)
)

// Maps 2-character state codes to full names
var	stateMap =  map[string]string {
	// USA
	"AK": "Alaska",
	"AL": "Alabama",
	"AR": "Arkansas",
	"AZ": "Arizona",
	"CA": "California",
	"CO": "Colorado",
	"CT": "Connecticut",
	"DC": "D.C.",
	"DE": "Delaware",
	"FL": "Florida",
	"GA": "Georgia",
	"HI": "Hawaii",
	"IA": "Iowa",
	"ID": "Idaho",
	"IL": "Illinois",
	"IN": "Indiana",
	"KS": "Kansas",
	"KY": "Kentucky",
	"LA": "Louisiana",
	"MA": "Massachusetts",
	"MD": "Maryland",
	"ME": "Maine",
	"MI": "Michigan",
	"MN": "Minnesota",
	"MO": "Missouri",
	"MS": "Mississippi",
	"MT": "Montana",
	"NC": "North Carolina",
	"ND": "North Dakota",
	"NE": "Nebraska",
	"NH": "New Hampshire",
	"NJ": "New Jersey",
	"NM": "New Mexico",
	"NV": "Nevada",
	"NY": "New York",
	"OH": "Ohio",
	"OK": "Oklahoma",
	"OR": "Oregon",
	"PA": "Pennsylvania",
	"RI": "Rhode Island",
	"SC": "South Carolina",
	"SD": "South Dakota",
	"TN": "Tennessee",
	"TX": "Texas",
	"UT": "Utah",
	"VA": "Virginia",
	"VT": "Vermont",
	"WA": "Washington",
	"WI": "Wisconsin",
	"WV": "West Virginia",
	"WY": "Wyoming", 

	// Canada
	"AB": "Alberta",
	"BC": "British Columbia",
	"MB": "Manitoba",
	"NB": "New Brunswick",
	"NL": "Newfoundland",
	"NS": "Novia Scotia",
	"NT": "Northwest Territories",
	"NU": "Nunavut",
	"ON": "Ontario",
	"PE": "Prince Edward Island",
	"QC": "Québec",
	"SK": "Saskatchewan",
	"YT": "Yukon",

	// Mexico
	"TB": "Tabasco",
	"AG": "Aguascalientes",
	"OA": "Oaxaca",
}

var reverseStateMap = map[string]string{}

var expansions = map[string]string {
	"S": "South",
	"W": "West",
	"N": "North",
	"E": "East",
	"Afb": "Air Force Base",
	"Bch": "Beach",
	"Brch": "Branch",
	"Ci": "City",
	"Cit": "City",
	"Crk": "Creek",
	"Ctr": "Center",
	"Cy": "City",
	"Depo": "Depot",
	"Fls": "Falls",
	"Fk": "Fork",
	"Forg": "Forge",
	"Frg": "Forge",
	"Ft": "Fort",
	"Ft.": "Fort",
	"Gdn": "Garden",
	//"Gr": TODO [] Great or Grand
	"Grv": "Grove",
	"Hgts": "Heights",
	"Hts": "Heights",
	"Jct": "Junction",
	"Lk": "Lake",
	"Mt": "Mount",
	"Mtn": "Mountain",
	"Pk": "Park",
	"Pnt": "Point",
	"Prt": "Port",
	"Rpds": "Rapids",
	"Rvr": "River",
	"Snta": "Santa",
	"Spgs": "Springs",
	"Spr": "Spring",
	"Sprs": "Springs",
	"St": "Saint",
	"St.": "Saint",
}

type CityState struct {
	City, State string
}

func init() {
	for code, name := range stateMap {
		reverseStateMap[name] = code
	}
}

func StateCode(name string) string {
	if _, has := stateMap[name]; has {
		return name
	}
	if code, has := reverseStateMap[name]; has {
		return code
	}
	return name
}

func StateName(code string) string {
	if name, has := stateMap[code]; has {
		return name
	}
	if _, has := reverseStateMap[code]; has {
		return code
	}
	return code
}

func ExpandCitySpelling(city string) string {
	names := strings.Split(ProperName(city), " ")
	for i, n := range names {
		r, ok := expansions[n]
		if ok {
			names[i] = r
		}
	}
	return strings.Join(names, " ")
}

func ProperName(s string) string {
	words := strings.Split(s, " ")
	var out []string
	for _, w := range words {
		if len(w) == 0 {
			continue
		}
		out = append(out, strings.Title(strings.ToLower(w)))
	}
	return strings.Join(out, " ")
}

func wikiProperName(s string) string {
	return strings.Replace(ProperName(s), " ", "_", -1)
}

func unwikiProperName(s string) string {
	return ProperName(strings.Replace(s, "_", " ", -1))
}

func GuessWikiUri1(cs CityState) (CityState, string) {
	name := CityState{ExpandCitySpelling(cs.City), StateName(cs.State)}
	return name, name.WikiUri()
}

func GuessWikiUri2(cs CityState) (CityState, string, error) {
	name, _ := GuessWikiUri1(cs)
	gname, guri, gerr := SearchGoogle(name)
	if gerr != nil {
		return name, "", gerr
	}
	return gname, guri, nil
}

func (cs CityState) String() string {
	return cs.City + ", " + cs.State
}

func (cs CityState) WikiUri() string {
	return wikiBaseUri + 
		wikiProperName(cs.City) + ",_" + wikiProperName(cs.State)
}

func ParseCityState(s string) (cs CityState) {
	m := wikiCityStateRe.FindStringSubmatch(s)
	if len(m) != 0 {
		cs.City = m[1]
		cs.State = m[2]
	}
	return 
}

