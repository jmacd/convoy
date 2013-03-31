package common

import "net/url"
import "regexp"
import "strings"

const (
	WikiHost              = "en.wikipedia.org"
	WikiBaseUri           = "/wiki/"
	WikiDisambiguationUri = "/wiki/Help:Disambiguation"
)

var (
	cityStateRe = regexp.MustCompile(`(.*), ([^,]+)`)
	separators  = []string{
		" ", // Space
		"–", // N-dash
		"—", // M-dash
		"―", // Figure-dash
	}
)

type CityState struct {
	City, State string
}

// Maps 2-character state codes to full names
var stateMap = map[string]string{
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
	"MX": "Mexico",
}

var reverseStateMap = map[string]string{}

var expansions = map[string][]string{
	"S":  []string{"South"},
	"So": []string{"North"},
	"W":  []string{"West"},
	"No": []string{"North"},
	"N":  []string{"North"},
	"E":  []string{"East"},

	"Afb":   []string{"Air Force Base"},
	"Ap":    []string{"Airport"},
	"Arprt": []string{"Airport"},
	"Bch":   []string{"Beach"},
	"Brdg":  []string{"Bridge"},
	"Brg":   []string{"Bridge"},
	"Brch":  []string{"Branch"},
	"Brk":   []string{"Brook"},
	"Blf":   []string{"Bluff"},
	"Blfs":  []string{"Bluffs"},
	"Ci":    []string{"City"},
	"Cit":   []string{"City"},
	"Ch":    []string{"Courthouse"},
	"Clg":   []string{"College"},
	"Crk":   []string{"Creek"},
	"Ctr":   []string{"Center"},
	"Ct":    []string{"Court"},
	"Cthse": []string{"Courthouse"},
	"Crt":   []string{"Court"},
	"Cy":    []string{"City"},
	"Depo":  []string{"Depot"},
	"Fk":    []string{"Fork"},
	"Fks":   []string{"Forks"},
	"Fls":   []string{"Falls"},
	"Forg":  []string{"Forge"},
	"Frg":   []string{"Forge"},
	"Ft":    []string{"Fort"},
	"Gdn":   []string{"Garden"},
	"Gr":    []string{"Great", "Grand"},
	"Grv":   []string{"Grove"},
	"Gln":   []string{"Glen"},
	"Hbr":   []string{"Harbor"},
	"Hse":   []string{"House"},
	"Hgts":  []string{"Heights"},
	"Hts":   []string{"Heights"},
	"Is":    []string{"Isle", "Island"},
	"Intl":  []string{"International"},
	"Jct":   []string{"Junction"},
	"Junct": []string{"Junction"},
	"Lk":    []string{"Lake"},
	"Mt":    []string{"Mount", "Mountain"},
	"Mtn":   []string{"Mountain"},
	"Pk":    []string{"Park"},
	"Pnt":   []string{"Point"},
	"Ps":    []string{"Pass"},
	"Pt":    []string{"Point"},
	"Ptr":   []string{"Point"},
	"Prtg":  []string{"Portage"},
	"Prt":   []string{"Port"},
	"Rdg":   []string{"Ridge"},
	"Rpds":  []string{"Rapids"},
	"Rvr":   []string{"River"},
	"Snta":  []string{"Santa"},
	"Spgs":  []string{"Springs"},
	"Spg":   []string{"Spring"},
	"Spr":   []string{"Spring"},
	"Sprs":  []string{"Springs"},
	"Sta":   []string{"Station"},
	"St":    []string{"Saint"},
	"Univ":  []string{"University"},
	"Wht":   []string{"White"},
	"Wks":   []string{"Works"},
	"Vly":   []string{"Valley"},
	"Vla":   []string{"Villa"},
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

func IsAStateName(name string) bool {
	_, has := reverseStateMap[name]
	return has
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

func Expand(n string) []string {
	s := strings.TrimRight(n, ".")
	if l, ok := expansions[s]; ok {
		return l
	}
	return []string{s}
}

func ExpandCitySpelling(city string) []string {
	names := strings.Split(ProperName(city), " ")
	exps := [][]string{[]string{}}
	for _, n := range names {
		l := Expand(n)
		var nexps [][]string
		for _, r := range l {
			for _, e := range exps {
				nexps = append(nexps, append(e, r))
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

func properNameFunc(in string, seps []string) string {
	if len(seps) == 0 {
		return strings.Title(in)
	}
	words := strings.Split(in, seps[0])
	var out []string
	for _, w := range words {
		if len(w) == 0 {
			continue
		}
		out = append(out, properNameFunc(w, seps[1:]))
	}
	return strings.Join(out, seps[0])
}

func ProperName(s string) string {
	return properNameFunc(strings.ToLower(s), separators)
}

func wikiProperName(s string) string {
	return strings.Replace(ProperName(s), " ", "_", -1)
}

func unwikiProperName(s string) string {
	return ProperName(strings.Replace(s, "_", " ", -1))
}

func WikiUrlToCityState(s string) (CityState, bool) {
	url, err := url.Parse(s)
	if err != nil {
		return CityState{}, false
	}
	if !strings.HasPrefix(url.Path, WikiBaseUri) {
		return CityState{}, false
	}
	cs := ParseCityState(unwikiProperName(url.Path[len(WikiBaseUri):]))
	if IsAStateName(cs.State) {
		return cs, true
	}
	return CityState{}, false
}

func GuessCityNames(cs CityState) (l []CityState) {
	cities := ExpandCitySpelling(cs.City)
	state := StateName(cs.State)
	for _, city := range cities {
		l = append(l, CityState{city, state})
	}
	return l
}

func (cs CityState) String() string {
	return cs.City + ", " + StateCode(cs.State)
}

func (cs CityState) WikiUri() string {
	return WikiBaseUri +
		wikiProperName(cs.City) + ",_" + wikiProperName(cs.State)
}

func (cs0 CityState) Equals(cs1 CityState) bool {
	return cs0.City == cs1.City && cs0.State == cs1.State
}

func ParseCityState(s string) (cs CityState) {
	m := cityStateRe.FindStringSubmatch(s)
	if len(m) != 0 {
		cs.City = m[1]
		cs.State = m[2]
	}
	return
}
