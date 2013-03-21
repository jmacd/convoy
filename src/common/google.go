package common

//import "bytes"
import "encoding/json"
import "errors"
import "log"
import "regexp"
import "net/url"
import "strings"

const (
	googleHost = "www.googleapis.com"
	googleBaseUri = "/customsearch/v1"

	// jmacd.llc
	googleCx = "018438991833677974599:jcnnzea70bm"
	googleKey = "AIzaSyDuCc1O8CPy7FmT6_Q6XfqYtdNcCqTEHDg"
	
	// josh.macdonald
	//googleCx = "009758624066673491715:ot7tq3iqsbq"
	//googleKey = "AIzaSyAoocCn9R3G1PH9DoOFwASBa4ONZHWIWrk"

)

var wikiLinkRe = regexp.MustCompile(`http://` + 
	regexp.QuoteMeta(WikiHost) + WikiBaseUri + `(.*)`)

var stripSiteRe = regexp.MustCompile(`(.*) site:.*`)

func CorrectCitySpelling(name CityState) (CityState, string, string, error) {
	spaceState := " " + name.State
	query := name.City + spaceState + " site:" + WikiHost
	googQuery := "?q=" + url.QueryEscape(query) + "&cx=" + 
		googleCx + "&key=" + googleKey + "&hl=en"
 	googXml, err := GetSecureUrl(googleHost, googleBaseUri, googQuery)
 	if err != nil {
		return CityState{}, "", "", nil
	}
	var res interface{}
	if err = json.Unmarshal(googXml, &res); err != nil {
 		log.Print("Google gave bad JSON: ", string(googXml))
		return CityState{}, "", "", nil
	}
	jso := res.(map[string]interface{})
	if gerror, has := jso["error"]; has {
		jsoError := gerror.(map[string]interface{})
		return CityState{}, "", "", errors.New(jsoError["message"].(string))
	}
	// TODO(jmacd): Create a Go type hierarchy instead of dealing w/ generics?
	// if true {
	// 	var buf bytes.Buffer
	// 	json.Indent(&buf, googXml, "", "\t")
 	// 	log.Print("Google JSON: ", string(buf.Bytes()))
        // }
	spellName := name
	if spell, has := jso["spelling"]; has {
		jsoSpell := spell.(map[string]interface{})
		if corrected, has := jsoSpell["correctedQuery"]; has {
			switch cv := corrected.(type) {
			case string:
				m := stripSiteRe.FindStringSubmatch(cv)
				if len(m) != 0 && strings.HasSuffix(m[1], spaceState) {
					spellName = CityState{m[1][:len(m[1]) - 
							len(spaceState)], name.State}
					//log.Println("Spelling", name, "->", spellName)
				}
			}
		}
	}
	var wikiNames []string
	if items, has := jso["items"]; has {
		jsoItems := items.([]interface{})
		for _, item := range jsoItems {
			jsoItem := item.(map[string]interface{})
			if jsoLink, has := jsoItem["link"]; has {
				switch link := jsoLink.(type) {
				case string:
					m := wikiLinkRe.FindStringSubmatch(link)
					if len(m) != 0 {
						wikiNames = append(wikiNames, m[1])
					}
				}
			}
		}
	}
	for _, wikiName := range wikiNames {
		unwikiName := unwikiProperName(wikiName)
		wikiCs := ParseCityState(unwikiName)
		if IsAStateName(wikiCs.State) {
			//log.Printf("Found city/state match: %s -> %s", spellName, wikiCs)
			return wikiCs, WikiBaseUri + wikiName, "geo-search", nil
		}
	}
	for _, wikiName := range wikiNames {
		//log.Printf("Found other match: %s -> %s", spellName, wikiCs)
		unwikiName := unwikiProperName(wikiName)
		city := spellName
		placeName, err := url.QueryUnescape(unwikiName)
		if err == nil {
			city = CityState{ProperName(placeName), spellName.State}
		}
		return city, WikiBaseUri + wikiName, "other-search", nil
	}
	return spellName, spellName.WikiUri(), "spell-search", nil
}
