// Interface for reading load information from any load board.

package boards

import "fmt"
import "time"

import "common"
import "scraper"

type LoadBoard interface {
	Init() error
	Read(chan<- scraper.Page)
}

type Load struct {
	ScrapeId    int64
	PickupDate  time.Time
	Origin      common.CityState
	Dest        common.CityState
	LoadType    string // "Full" or "Partial" or ?
	Length      int
	Weight      int
	Equipment   string
	Price       int
	Stops       int
	Phone       string
}

func (l *Load) String() string {
	return fmt.Sprintf("[%d] %v %v -> %v %v %v %v %v %v %v %v",
		l.ScrapeId, l.PickupDate.Format(common.SqlDateFmt),
		l.Origin, l.Dest, l.LoadType, l.Length, l.Weight, 
		l.Equipment, l.Price, l.Stops, l.Phone)
}
