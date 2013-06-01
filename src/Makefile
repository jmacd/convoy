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
LFILES = loadtool.go $(GOFILES)
OFILES = osrmtool.go $(GOFILES)

BINDIR = ../bin

TARGETS = \
	$(BINDIR)/convoy \
	$(BINDIR)/geotool \
	$(BINDIR)/loadtool \
	$(BINDIR)/maptool \
	$(BINDIR)/osrmtool

%.pb.go: %.proto
	protoc --go_out=. $<

all: dirs $(TARGETS)

$(BINDIR)/convoy: $(CFILES)
	go build -o $(BINDIR)/convoy convoy.go

$(BINDIR)/maptool: $(MFILES)
	go build -o $(BINDIR)/maptool maptool.go

$(BINDIR)/geotool: $(GFILES)
	go build -o $(BINDIR)/geotool geotool.go

$(BINDIR)/loadtool: $(LFILES)
	go build -o $(BINDIR)/loadtool loadtool.go

$(BINDIR)/osrmtool: $(OFILES)
	go build -o $(BINDIR)/osrmtool osrmtool.go

dirs:
	go install boards
	go install common
	go install data
	go install geo
	go install graph
	go install maps
	go install scraper

test: test_common test_geo test_graph test_scraper

test_common:
	go test common

test_geo:
	go test geo

test_graph:
	go test graph

test_scraper:
	go test scraper
