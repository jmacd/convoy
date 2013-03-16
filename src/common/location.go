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

var expansions = map[string][]string {
	"S": []string{"South"},
	"W": []string{"West"},
	"N": []string{"North"},
	"E": []string{"East"},
	"Afb": []string{"Air Force Base"},
	"Ap": []string{"Airport"},
	"Bch": []string{"Beach"},
	"Brch": []string{"Branch"},
	"Ci": []string{"City"},
	"Cit": []string{"City"},
	"Crk": []string{"Creek"},
	"Ctr": []string{"Center"},
	"Cy": []string{"City"},
	"Depo": []string{"Depot"},
	"Fk": []string{"Fork"},
	"Fls": []string{"Falls"},
	"Forg": []string{"Forge"},
	"Frg": []string{"Forge"},
	"Ft": []string{"Fort"},
	"Ft.": []string{"Fort"},
	"Gdn": []string{"Garden"},
	"Gr": []string{"Great", "Grand"},
	"Grv": []string{"Grove"},
	"Hbr": []string{"Harbor"},
	"Hgts": []string{"Heights"},
	"Hts": []string{"Heights"},
	"Intl": []string{"International"},
	"Jct": []string{"Junction"},
	"Lk": []string{"Lake"},
	"Mt": []string{"Mount", "Mountain"},
	"Mtn": []string{"Mountain"},
	"Pk": []string{"Park"},
	"Pnt": []string{"Point"},
	"Prt": []string{"Port"},
	"Rpds": []string{"Rapids"},
	"Rvr": []string{"River"},
	"Snta": []string{"Santa"},
	"Spgs": []string{"Springs"},
	"Spr": []string{"Spring"},
	"Sprs": []string{"Springs"},
	"Sta": []string{"Station"},
	"St": []string{"Saint"},
	"St.": []string{"Saint"},
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

func ExpandCitySpelling(city string) []string {
	names := strings.Split(ProperName(city), " ")
	exps := [][]string{[]string{}}
	for _, n := range names {
		l, ok := expansions[n]
		if !ok {
			l = []string{n}
		}
		var nexps [][]string
		for _, r := range l {
			for _, n := range exps {
				nexps = append(nexps, append(n, r))
			}
		}
		exps = nexps
	}
	res := []string{}
	for _, e := range exps {
		res = append(res, strings.Join(e, " "))
	}
	return res
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

func GuessCityNames(cs CityState) (l []CityState) {
	cities := ExpandCitySpelling(cs.City)
	state := StateName(cs.State)
	for _, city := range cities {
		l = append(l, CityState{city, state})
	}
	return l
}

func CorrectCitySpelling(name CityState) (CityState, string, error) {
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
