package common

import "log"
import "encoding/json"
import "strings"
import "regexp"

const (
	googleHost = "www.googleapis.com"
	googleBaseUri = "/customsearch/v1"
	googleCx = "018438991833677974599:jcnnzea70bm"
	googleKey = "AIzaSyDuCc1O8CPy7FmT6_Q6XfqYtdNcCqTEHDg"
)

var googleQueryRe = regexp.MustCompile(`"([^"]+)" ([^ ]+)`)

func SearchGoogle(name CityState) (CityState, string, error) {
	query := "\"" + name.City + "\" " + name.State
	googQuery := "?q=" + strings.Replace(query, " ", "+", -1) +
		"&cx=" + googleCx + "&key=" + googleKey +
		"&hl=en"
 	googXml, err := GetSecureUrl(googleHost, googleBaseUri, googQuery)
 	if err != nil {
		return CityState{}, "", nil
	}
	var res interface{}
	if err = json.Unmarshal(googXml, &res); err != nil {
 		log.Print("Google gave bad JSON: ", string(googXml))
		return CityState{}, "", nil
	}
	jso := res.(map[string]interface{})
	if spell, has := jso["spelling"]; has {
		jsoSpell := spell.(map[string]interface{})
		if corrected, has := jsoSpell["correctedQuery"]; has {
			switch cv := corrected.(type) {
			case string:
				m := googleQueryRe.FindStringSubmatch(cv)
				if len(m) != 0 {
					cs := CityState{m[1], m[2]}
					return cs, cs.WikiName(), nil
				}
			}
		}
	}
	return name, "", nil
}
