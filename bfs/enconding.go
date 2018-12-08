package bfs

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
)

type Edge struct {
	SrcID int64
	DstID int64
}

type Encoder interface {
	Encode(Edge) error
	Close() error
}

type Decoder interface {
	Decode() (Edge, error)
}

func DecodeAll(d Decoder) ([]Edge, error) {
	edges := make([]Edge, 0, 5000)
	for edge, err := d.Decode(); err != io.EOF; edge, err = d.Decode() {
		if err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

type TextEncoder struct {
	writer io.Writer
}

func NewTextEncoder(writer io.Writer) *TextEncoder {
	return &TextEncoder{
		writer: writer,
	}
}

func (te *TextEncoder) Encode(edge Edge) {
	te.writer.Write([]byte(fmt.Sprintf("%v\t%v\n", edge.SrcID, edge.DstID)))
}

func (te *TextEncoder) Close() error {
	return nil
}

type TextDecoder struct {
	scanner bufio.Scanner
}

func NewTextDecoder(reader io.Reader) *TextDecoder {
	scanner := bufio.NewScanner(reader)
	gob.Register(Edge{})
	return &TextDecoder{
		scanner: *scanner,
	}
}

// This function allows us to do allocation free parsing of each lines
func parseBytes(bts []byte) int64 {
	var x int64
	for _, c := range bts {
		x = x*10 + int64(c-'0')
	}
	return x
}

func (te *TextDecoder) Decode() (Edge, error) {
	if ok := te.scanner.Scan(); !ok {
		if te.scanner.Err() != nil {
			return Edge{}, te.scanner.Err()
		}
		return Edge{}, io.EOF
	}
	line := te.scanner.Bytes()
	i := bytes.Index(line, []byte("\t"))
	// Use this rather than strconv.ParseInt becuase that requires
	// converting to strings, and thus allocating strings
	srcID := parseBytes(line[:i])
	dstID := parseBytes(line[i+1:])
	return Edge{
		SrcID: srcID,
		DstID: dstID,
	}, nil
}

func (te *TextDecoder) DecodeInts() (int64, int64, error) {
	if ok := te.scanner.Scan(); !ok {
		if te.scanner.Err() != nil {
			return 0, 0, te.scanner.Err()
		}
		return 0, 0, io.EOF
	}
	line := te.scanner.Bytes()
	i := bytes.Index(line, []byte("\t"))
	// Use this rather than strconv.ParseInt becuase that requires
	// converting to strings, and thus allocating strings
	srcID := parseBytes(line[:i])
	dstID := parseBytes(line[i+1:])
	return srcID, dstID, nil
}

func (te *TextDecoder) Close() error {
	return nil
}

type BinaryEncoder struct {
	compressor *gzip.Writer
	gobEncoder *gob.Encoder
}

func NewBinaryEncoder(writer io.Writer, compress bool) *BinaryEncoder {
	if compress {
		compressor := gzip.NewWriter(writer)
		gobEncoder := gob.NewEncoder(compressor)
		return &BinaryEncoder{
			compressor: compressor,
			gobEncoder: gobEncoder,
		}
	}
	gobEncoder := gob.NewEncoder(writer)
	return &BinaryEncoder{
		gobEncoder: gobEncoder,
	}
}

func (be *BinaryEncoder) Encode(edge Edge) {
	be.gobEncoder.Encode(edge)
}

func (be *BinaryEncoder) Close() error {
	if be.compressor != nil {
		return be.compressor.Close()
	}
	return nil
}

type BinaryDecoder struct {
	decodeEdges  []Edge
	indexChan    chan int
	decompressor *gzip.Reader
	gobDecoder   *gob.Decoder
}

func NewBinaryDecoder(reader io.Reader, decompress bool) (*BinaryDecoder, error) {
	decodeEdges := make([]Edge, 1000)
	indexChan := make(chan int, 1000)
	for i := 0; i < 1000; i++ {
		indexChan <- i
	}

	if decompress {
		decompressor, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		gobDecoder := gob.NewDecoder(decompressor)
		return &BinaryDecoder{
			decodeEdges:  decodeEdges,
			indexChan:    indexChan,
			decompressor: decompressor,
			gobDecoder:   gobDecoder,
		}, nil
	}
	gobDecoder := gob.NewDecoder(reader)
	return &BinaryDecoder{
		decodeEdges: decodeEdges,
		indexChan:   indexChan,
		gobDecoder:  gobDecoder,
	}, nil
}

func (bd *BinaryDecoder) Decode() (Edge, error) {
	var edge Edge
	err := bd.gobDecoder.Decode(&edge)
	if err != nil {
		return Edge{}, err
	}
	return edge, nil
}

func (bd *BinaryDecoder) DecodeInts() (int64, int64, error) {
	ind := <-bd.indexChan
	err := bd.gobDecoder.Decode(&bd.decodeEdges[ind])
	srcID := int64(bd.decodeEdges[ind].SrcID)
	dstID := int64(bd.decodeEdges[ind].DstID)
	bd.indexChan <- ind
	if err != nil {
		return 0, 0, err
	}
	return srcID, dstID, nil
}

func (bd *BinaryDecoder) Close() error {
	if bd.decompressor != nil {
		return bd.decompressor.Close()
	}
	return nil
}
