package maps

import "bytes"
import "compress/zlib"
import "encoding/binary"
import "errors"
import "fmt"
import "io"
import "io/ioutil"
import "log"
import "code.google.com/p/goprotobuf/proto"

import "proto/osm"

func readFixed(f io.Reader, s int32) ([]byte, error) {
	buf, err := ioutil.ReadAll(&io.LimitedReader{f, int64(s)})
	if err != nil {
		return nil, err
	}
	if len(buf) != int(s) {
		return nil, errors.New(
			fmt.Sprintln("Insufficient read:", len(buf), s))
	}
	return buf, nil
}

func ReadMap(f io.Reader) error {
	var nread int64
	for {
		// Read the next blob header size
		hsizeb, err := readFixed(f, 4)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		nread += int64(4)

		// Decode four bytes
		var hsize int32
		binary.Read(bytes.NewReader(hsizeb), binary.BigEndian, &hsize)

		// Read the next blob header
		headb, err := readFixed(f, hsize)
		if err != nil {
			return err
		}
		nread += int64(hsize)

		// Unmarshal the header
		var bh osm.BlobHeader
		if err = proto.Unmarshal(headb, &bh); err != nil {
			return err
		}
		
		// Read the blob itself
		bsize := bh.GetDatasize()
		if bsize <= 0 {
			return errors.New("Zero byte blob; quitting")
		}
		blobb, err := readFixed(f, bsize)
		if err != nil {
			return err
		}
		nread += int64(bsize)
		
		// Unmarshal the blob
		var blob osm.Blob
		if err = proto.Unmarshal(blobb, &blob); err != nil {
			return err
		}
		enc := "unknown"
		var data []byte
		switch {
		case blob.Raw != nil:
			enc = "raw"
			data = blob.Raw
		case blob.ZlibData != nil:
			enc = "zlib"
			zr, err := zlib.NewReader(bytes.NewReader(blob.ZlibData))
			if err != nil {
				return err
			}
			defer zr.Close()
			if data, err = readFixed(zr, blob.GetRawSize()); err != nil {
				return err
			}
		case blob.LzmaData != nil:
			enc = "lzma"
		}
		if data == nil {
			log.Fatalln("Unsupported OSM data encoding", enc)
		}

		log.Printf("Read a blob %s type %s size %d / %d", 
			bh.GetType(), enc, bsize, blob.GetRawSize())
		switch bh.GetType() {
		}
	}
	log.Println("Finished reading", nread, "bytes")
	return nil
}
