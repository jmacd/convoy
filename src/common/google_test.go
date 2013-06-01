package common

import "io/ioutil"

import "testing"

const (
	sampleGoogleJsonFile = "./google.json"
)

func TestCorrection(t *testing.T) {
	sample, err := ioutil.ReadFile(sampleGoogleJsonFile)
	if err != nil {
		t.Errorf("Can't read %v", sampleGoogleJsonFile)
	}
	corrected, uri, desc, err := processGoogleResult(ParseCityState("Oiowa City, Iowa"), sample)
	// Not a great result....
	if !corrected.Equals(ParseCityState("Oelwein, Iowa")) {
		t.Errorf("Incorrect city: %v", corrected)
	}
	if uri != "/wiki/Oelwein,_Iowa" || desc != "geo-search" {
		t.Errorf("Incorrect uri/desc: %v %v", uri, desc)
	}
	if err != nil {
		t.Errorf("google error")
	}
}