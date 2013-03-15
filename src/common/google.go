package common

import "bytes"
import "encoding/json"
import "errors"
import "log"
import "regexp"
import "strings"

const (
	googleHost = "www.googleapis.com"
	googleBaseUri = "/customsearch/v1"
	googleCx = "018438991833677974599:jcnnzea70bm"
	googleKey = "AIzaSyDuCc1O8CPy7FmT6_Q6XfqYtdNcCqTEHDg"
)

var wikiLinkRe = regexp.MustCompile(`http://` + 
	regexp.QuoteMeta(WikiHost) + wikiBaseUri + `(.*)`)

func SearchGoogle(name CityState) (CityState, string, error) {
	spaceState := " " + name.State
	query := name.City + spaceState
	googQuery := "?q=" + strings.Replace(query, " ", "+", -1) +
		"&cx=" + googleCx + "&key=" + googleKey +
		"&hl=en" + "&siteSearch=" + WikiHost
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
	if gerror, has := jso["error"]; has {
		jsoError := gerror.(map[string]interface{})
		return CityState{}, "", errors.New(jsoError["message"].(string))
	}
	// TODO(jmacd): Create a Go type hierarchy instead of dealing w/ generics?
	if true {
		var buf bytes.Buffer
		json.Indent(&buf, googXml, "", "\t")
 		log.Print("Google JSON: ", string(buf.Bytes()))
        }
	spellName := name
	if spell, has := jso["spelling"]; has {
		jsoSpell := spell.(map[string]interface{})
		if corrected, has := jsoSpell["correctedQuery"]; has {
			switch cv := corrected.(type) {
			case string:
				log.Println("Spelling", cv)
				if strings.HasSuffix(cv, spaceState) {
					spellName := CityState{cv[:len(cv) - len(spaceState)], name.State}
					log.Println("Spelling", name, " -> ", spellName) 
				}
			}
		}
	}
	var cityNames []string
	var otherNames []string
	if items, has := jso["items"]; has {
		jsoItems := items.([]interface{})
		for _, item := range jsoItems {
			jsoItem := item.(map[string]interface{})
			if jsoLink, has := jsoItem["link"]; has {
				switch link := jsoLink.(type) {
				case string:
					m := wikiLinkRe.FindStringSubmatch(link)
					if len(m) != 0 {
						wikiName := m[1]
						unwikiName := unwikiProperName(wikiName)
						if unwikiName == spellName.String() {
							//log.Println("Found exact city match: ", spellName)
							return spellName, spellName.WikiUri(), nil
						}
						if strings.HasSuffix(unwikiName, spaceState) {
							cityNames = append(cityNames, 
								unwikiName[:len(unwikiName) - len(spaceState) - 1])
						} else {
							otherNames = append(otherNames, unwikiName)
						}
					}
				}
			}
		}
	}
	//log.Printf("Found city names %q", cityNames)
	//log.Printf("Found other names %q", otherNames)
	if len(cityNames) > 0 {
		cs := CityState{cityNames[0], name.State}
		return cs, cs.WikiUri(), nil
	}
	if len(otherNames) > 0 {
		return spellName, wikiBaseUri + wikiProperName(otherNames[0]), nil
	}
	return spellName, spellName.WikiUri(), nil
}
