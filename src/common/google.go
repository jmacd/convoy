package common

// import "errors"
// import "log"
// import "code.google.com/p/go.net/html/atom"

const (
	googleHost = "www.googleapis.com"
	googleBaseUri = "/customsearch/v1"
)


	// 	googQuery := "?q=" + strings.Replace(exp.City + " " + 
	// 		exp.State + " site:en.wikipedia.org", " ", "+", -1)
	// 	googXml, err := GetUrl(googleHost, googleBaseUri, googQuery)

	// 	if err == nil {
	// 		var wikiNames []string
	// 			err := scraper.ParseXml(googXml, atom.A, "href",
	// 			func (value string) func (text string) {
	// 			m := wikiUrlRe.FindStringSubmatch(value)
	// 			if len(m) != 0 {
	// 				return func (text string) {
	// 					wikiNames = append(wikiNames, 
	// 						unwikiProperName(m[1]))
	// 				}
	// 			}
	// 			return nil
	// 		})
	// 		if err != nil {
	// 			log.Print("Could not parse Google result:", googQuery)
	// 		}
	// 		log.Print("Google: ", string(googXml))
	// 		if len(wikiNames) != 0 {
	// 			wcs := ParseCityState(wikiNames[0])
	// 			if len(wcs.City) != 0 && wcs.State == exp.State {
	// 				return wcs, wikiBaseUri + wikiNames[0], nil
	// 			}
	// 		}
	// 	}
	// }
