package main

import "regexp"
import "log"
//import "io/ioutil"

const (
	// escapedRegexp = `[a-zA-Z0-9$&#;,]`
	// pageRegexp = `javascript:__doPostBack\(` + escapedRegexp + 
	// 	`+Page` + escapedRegexp + `+\)`
	identifierRegexp = `[a-zA-Z0-9$]+`
	// actionRegexp = `__doPostBack\('(` + identifierRegexp + `)','(` +
	// 	identifierRegexp + `)\)`
	actionRegexp = `__doPostBack\('(` + identifierRegexp + `)','(` +
		identifierRegexp + `)'\)`
)

func main() {
	s := "__doPostBack('ctl00$ContentPlaceHolder1$GridView1','Page$3')"
	re := regexp.MustCompile(actionRegexp)
	matches := re.FindAllString(s, -1)
	for _, m := range matches {
		log.Print("Match: ", m)
	}
}