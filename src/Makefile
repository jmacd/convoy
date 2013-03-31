GOFILES = \
	boards/loadboard.go \
	boards/trulos.go \
	boards/util.go \
	common/cncrntzr.go \
	common/common.go \
	common/google.go \
	common/location.go \
	data/db.go \
	data/fix.go \
	data/model.go \
	geo/kdtree.go \
	geo/point.go \
	geo/pointconv.go \
	graph/sssp.go \
	maps/osmreader.go \
	proto/osm/fileformat.pb.go \
	proto/osm/osmformat.pb.go \
	scraper/browser.go \
	scraper/scrape.go \
	scraper/xml.go

CFILES = convoy.go $(GOFILES)
MFILES = maptool.go $(GOFILES)
GFILES = geotool.go $(GOFILES)

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

test: test_common test_geo

test_common:
	go test -i common
	go test common

test_geo:
	go test -i geo
	go test geo
