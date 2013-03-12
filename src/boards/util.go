package boards

import "log"
import "regexp"
import "strconv"

var (
	httpColonRe *regexp.Regexp = regexp.MustCompile("https?:/")
	fpnumRe     *regexp.Regexp = regexp.MustCompile(`^$?([0-9]+(?:\.[0-9]+)?)`)
)

func HijackExternalRefs(data []byte) []byte {
	return httpColonRe.ReplaceAll(data, []byte{})
}

func ParseLeadingInt(s string) int {
	if len(s) == 0 {
		return 0
	}
	m := fpnumRe.FindString(s)
	if len(m) != 0 {
		f, _ := strconv.ParseFloat(m, 64)
		return int(f)
	}
	log.Print("Could not parse number: ", s)
	return 0
}
