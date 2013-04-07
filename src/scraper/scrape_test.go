package scraper

import "testing"
import "time"

import "common"

type testDate struct {
	start, finish, date string
}

var dateTests = []testDate{
	{"2013-04-01 23:00:01","2013-04-02 01:00:01", "2013-04-02"},
	{"2013-04-01 10:00:01","2013-04-01 11:00:01", "2013-04-01"},
	{"2013-04-01 10:00:01","2013-04-01 11:00:01", "2013-04-01"},
	{"2013-04-01 12:00:00","2013-04-02 12:00:00", "2013-04-02"},
	{"2013-04-01 10:00:00","2013-04-02 00:00:00", "2013-04-01"},
	{"2013-04-02 01:00:00","2013-04-02 02:00:00", "2013-04-02"},
}

func TestScrapeDate(t *testing.T) {
	for _, dt := range dateTests {
		st, err := time.Parse(common.SqlDateFmt, dt.start)
		if err != nil {
			t.Error("bad date: ", err)
		}
		ft, err := time.Parse(common.SqlDateFmt, dt.finish)
		if err != nil {
			t.Error("bad date: ", err)
		}
		s := Scrape{0, st, ft}
		d := s.Date()
		ds := d.Format(common.LoadDateFmt)
		if ds != dt.date {
			t.Error("date mismatch: ", ds, "!=", dt.date)
		}
	}
}
