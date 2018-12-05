package bfs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TestString string = `2895778917	2133463915
2895778917	2133955112
2895778917	2140475752
2895778917	2144671150
2895778917	2165881478
2895778917	2168756686
2895778917	2317654055
2895778921	2017551946
2895778921	2345296451
2895778921	2471498311
`

var testEdges []Edge = []Edge{
	Edge{SrcID: 2895778917, DstID: 2133463915},
	Edge{SrcID: 2895778917, DstID: 2133955112},
	Edge{SrcID: 2895778917, DstID: 2140475752},
	Edge{SrcID: 2895778917, DstID: 2144671150},
	Edge{SrcID: 2895778917, DstID: 2165881478},
	Edge{SrcID: 2895778917, DstID: 2168756686},
	Edge{SrcID: 2895778917, DstID: 2317654055},
	Edge{SrcID: 2895778921, DstID: 2017551946},
	Edge{SrcID: 2895778921, DstID: 2345296451},
	Edge{SrcID: 2895778921, DstID: 2471498311}}

func TestTextEncoder(t *testing.T) {
	var testWriter bytes.Buffer
	encoder := NewTextEncoder(&testWriter)
	for _, edge := range testEdges {
		encoder.Encode(edge)
	}
	encoder.Close()
	assert.Equal(t, TestString, testWriter.String())
	testFile := bytes.NewBuffer(testWriter.Bytes())

	decoder := NewTextDecoder(testFile)
	decodedEdges, err := DecodeAll(decoder)
	assert.NoError(t, err)
	assert.Equal(t, testEdges, decodedEdges)

}

func TestTextDecoder(t *testing.T) {
	testFile := bytes.NewBuffer([]byte(TestString))

	decoder := NewTextDecoder(testFile)
	decodedEdges, err := DecodeAll(decoder)
	assert.NoError(t, err)
	assert.Equal(t, testEdges, decodedEdges)
}

func TestBinaryEncodeDecode(t *testing.T) {
	t.Run("No Compression", func(t *testing.T) {
		var testWriter bytes.Buffer
		encoder := NewBinaryEncoder(&testWriter, false)
		for _, edge := range testEdges {
			encoder.Encode(edge)
		}
		testFile := bytes.NewBuffer(testWriter.Bytes())
		encoder.Close()

		decoder, err := NewBinaryDecoder(testFile, false)
		defer decoder.Close()
		assert.NoError(t, err)
		decodedEdges, err := DecodeAll(decoder)
		assert.NoError(t, err)
		assert.Equal(t, testEdges, decodedEdges)
	})
	t.Run("With Compression", func(t *testing.T) {
		var testWriter bytes.Buffer
		encoder := NewBinaryEncoder(&testWriter, true)
		for _, edge := range testEdges {
			encoder.Encode(edge)
		}
		encoder.Close()
		testFile := bytes.NewBuffer(testWriter.Bytes())

		decoder, err := NewBinaryDecoder(testFile, true)
		defer decoder.Close()
		assert.NoError(t, err)
		decodedEdges, err := DecodeAll(decoder)
		assert.NoError(t, err)
		assert.Equal(t, testEdges, decodedEdges)
	})
}
