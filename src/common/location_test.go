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