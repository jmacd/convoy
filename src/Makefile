PFILES = \
	proto/osm/fileformat.pb.go \
	proto/osm/osmformat.pb.go

CFILES = \
	convoy.go \
	boards/common.go \
	boards/loadboard.go \
	boards/trulos.go \
	scraper/scrape.go

MFILES = \
	maptool.go \
	maps/osmreader.go \
	$(PFILES)

BINDIR = ../bin

TARGETS = \
	$(BINDIR)/convoy \
	$(BINDIR)/maptool

%.pb.go: %.proto
	protoc --go_out=. $<

all: $(TARGETS)

$(BINDIR)/convoy: $(CFILES)
	go build -o $(BINDIR)/convoy convoy.go

$(BINDIR)/maptool: $(MFILES)
	go build -o $(BINDIR)/maptool maptool.go
