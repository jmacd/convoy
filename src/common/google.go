package common

import "encoding/json"
import "errors"
import "log"
import "regexp"
import "net/url"
import "strings"

// TODO improve this logic. first, need to record query results.

const (
	googleHost    = "www.googleapis.com"
	googleBaseUri = "/customsearch/v1"

	// jmacd.llc
	googleCx  = "018438991833677974599:jcnnzea70bm"
	googleKey = "AIzaSyDuCc1O8CPy7FmT6_Q6XfqYtdNcCqTEHDg"

	// josh.macdonald
	//googleCx = "009758624066673491715:ot7tq3iqsbq"
	//googleKey = "AIzaSyAoocCn9R3G1PH9DoOFwASBa4ONZHWIWrk"

)

var wikiLinkRe = regexp.MustCompile(`http://` +
	regexp.QuoteMeta(WikiHost) + WikiBaseUri + `(.*)`)

var stripSiteRe = regexp.MustCompile(`(.*) site:.*`)

type JsonGoogle struct {
	Kind string
	Url *JsonGoogleUrl
	Queries *JsonGoogleQuery
	SearchInformation *JsonGoogleSearchInformation
	Spelling *JsonGoogleSpelling
	Items []*JsonGoogleItem
	Error *JsonGoogleError
}

type JsonGoogleUrl struct {
	Type string
	Template string
}

type JsonGoogleQuery struct {
	Request []JsonGoogleRequest
}

type JsonGoogleSpelling struct {
	CorrectedQuery string
}

type JsonGoogleSearchInformation struct {
	SearchTime float64
	TotalResults string
}

type JsonGoogleItem struct {
	Kind string
	Title string
	Link string
	Snippet string
}

type JsonGoogleRequest struct {
	SearchTerms string
	Count int
	StartIndex int
}

type JsonGoogleError struct {
	Message string
}

func CorrectCitySpelling(name CityState) (CityState, string, string, error) {
	spaceState := " " + name.State
	query := name.City + spaceState + " site:" + WikiHost
	googQuery := "?q=" + url.QueryEscape(query) + "&cx=" +
		googleCx + "&key=" + googleKey + "&hl=en"
	googXml, err := GetSecureUrl(googleHost, googleBaseUri, googQuery)
	if err != nil {
		return CityState{}, "", "", nil
	}
	return processGoogleResult(name, googXml)
}

func processGoogleResult(name CityState, xml []byte) (CityState, string, string, error) {
	spaceState := " " + name.State
	var res JsonGoogle
	if err := json.Unmarshal(xml, &res); err != nil {
		log.Print("Google gave bad JSON: ", string(xml))
		return CityState{}, "", "", nil
	}
	if res.Error != nil {
		return CityState{}, "", "", errors.New(res.Error.Message)
	}
	spellName := name
	if res.Spelling != nil && res.Spelling.CorrectedQuery != "" {
		m := stripSiteRe.FindStringSubmatch(res.Spelling.CorrectedQuery)
		if len(m) != 0 && strings.HasSuffix(m[1], spaceState) {
			spellName = CityState{m[1][:len(m[1])-
					len(spaceState)], name.State}
			//log.Println("Spelling", name, "->", spellName)
		}
	}
	var wikiNames []string
	for _, item := range res.Items {
		if item.Link != "" {
			m := wikiLinkRe.FindStringSubmatch(item.Link)
			if len(m) != 0 {
				wikiNames = append(wikiNames, m[1])
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
