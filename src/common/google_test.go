package common

import "encoding/json"
import "io/ioutil"
import "log"

import "testing"

const (
	sampleGoogleJsonFile = "./google.json"
)

func TestCorrection(t *testing.T) {
	sample, err := ioutil.ReadFile(sampleGoogleJsonFile)
	if err != nil {
		t.Errorf("Can't read %v", sampleGoogleJsonFile)
	}


		log.Print("Google gave bad JSONGoogle: ", string(sample))
	} else {
		// log.Print("JSONGoogle: ", test)
		// log.Print("JSONGoogleUrl: ", test.Url)
		// log.Print("JSONGoogleQueries: ", test.Queries)
		// log.Print("JSONGoogleSearchInfo: ", test.SearchInformation)
		// log.Print("JSONGoogleSpelling: ", test.Spelling)
		// for _, i := range test.Items {
		// 	log.Print("JSONGoogleItem: ", i)
		// }
	}
	corrected, uri, desc, err := processGoogleResult(ParseCityState("Oiowa City, Iowa"), sample)
	t.Errorf("Bio1 %v %v %v %v", corrected, uri, desc, err)
}