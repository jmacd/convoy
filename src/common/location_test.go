package common

import "sort"
import "testing"

type expCheck struct {
	in string
	out []string
}

var expExpect = []expCheck{
	{ "S Bend", []string{"South Bend"} },
	{ "Gr Neck", []string{"Great Neck", "Grand Neck"} },
	{ "Gdn Grove", []string{"Garden Grove"} },
}

func TestExpansion(t *testing.T) {
	for _, ec := range expExpect {
		r := ExpandCitySpelling(ec.in)
		sort.Strings(r)
		sort.Strings(ec.out)
		if len(r) != len(ec.out) {
			t.Errorf("Wrong number of results %q != %q", r, ec.out)
		} else {
			for i := 0; i < len(r); i++ {
				if r[i] != ec.out[i] {
					t.Errorf("%q != !q", r[i], ec.out[i])
				}
			}
		}
	}
}

var propExpect = [][2]string {
	[2]string{"The-intl airport", "The-Intl Airport"},
	[2]string{"What EVER", "What Ever"},
	[2]string{"Hartsfield–jackson Atlanta International Airport", 
		  "Hartsfield–Jackson Atlanta International Airport"},
}

func TestProper(t *testing.T) {
	for _, ep := range propExpect {
		r := ProperName(ep[0])
		if r != ep[1] {
			t.Errorf("Bad proper name %q != %q", r, ep[1])
		} 
	}
}

var cityExpect = [][2]string {
	[2]string{"http://en.wikipedia.org/wiki/Foo,_Wisconsin", "Foo, WI"},
	[2]string{"/wiki/Bar,_California", "Bar, CA"},
	[2]string{"/wiki/Place_In_The_Sun", ""},
	[2]string{"#Foo", ""},
}

func TestWikiToCity(t *testing.T) {
	for _, ce := range cityExpect {
		r, valid := WikiUrlToCityState(ce[0])
		var e string
		if valid {
			e = r.String()
		}
		if ce[1] != e {
			t.Errorf("City URL incorrect: %q %q %q", ce[0], ce[1], e)
		}
	}
}
