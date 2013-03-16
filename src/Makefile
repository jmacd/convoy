PFILES = \
	proto/osm/fileformat.pb.go \
	proto/osm/osmformat.pb.go

CFILES = \
	convoy.go \
	common/common.go \
	common/coordinates.go \
	common/google.go \
	common/location.go \
	data/db.go \
	boards/util.go \
	boards/loadboard.go \
	boards/trulos.go \
	scraper/browser.go \
	scraper/scrape.go

MFILES = \
	maptool.go \
	maps/osmreader.go \
	$(PFILES)

GFILES = \
	geotool.go \
	common/common.go \
	common/coordinates.go \
	common/google.go \
	common/location.go \
	data/db.go \
	data/fix.go \
	scraper/browser.go \
	scraper/xml.go

BINDIR = ../bin

TARGETS = \
	$(BINDIR)/convoy \
	$(BINDIR)/maptool \
	$(BINDIR)/geotool

%.pb.go: %.proto
	protoc --go_out=. $<

all: test $(TARGETS)

$(BINDIR)/convoy: $(CFILES)
	go build -o $(BINDIR)/convoy convoy.go

$(BINDIR)/maptool: $(MFILES)
	go build -o $(BINDIR)/maptool maptool.go

$(BINDIR)/geotool: $(GFILES)
	go build -o $(BINDIR)/geotool geotool.go

test: test_common

test_common:
	go test common
