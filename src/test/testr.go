package main

import "regexp"
import "log"
import "io/ioutil"

const (
	escapedRegexp = `[a-zA-Z0-9$&#;,]`
	pageRegexp = `javascript:__doPostBack\(` + escapedRegexp + 
		`+Page` + escapedRegexp + `+\)`
)

func main() {
        contents, err := ioutil.ReadFile("trulos.html")
	if err != nil {
		log.Print("Can't read file")
		return
	}
	re := regexp.MustCompile(pageRegexp)
	matches := re.FindAll(contents, -1)
	for _, m := range matches {
		log.Print("Match: ", string(m))
	}
}