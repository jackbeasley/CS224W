package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"./bfs"

	"net/http"
	_ "net/http/pprof"
)

const (
	srcHashDir    = "mag_src_hash"
	dstHashDir    = "mag_dst_hash"
	errPaperIDArg = "You must provide an integer paper ID as an argument"
)

func main() {
	var followOut = flag.Bool("out", false, "Follow out links (i.e. cited papers)")
	var followIn = flag.Bool("in", false, "Follow in links (i.e. paper who cite)")
	var levels = flag.Int("levels", 1, "How many levels to BFS out")
	var bloom = flag.Bool("bloom", false, "Use a bloom filter for really big BFSs")
	var fdLimit = flag.Int("fdLimit", 20, "How many open files to use")
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	paperIDStr := flag.Arg(0)
	if paperIDStr == "" {
		fmt.Println(errPaperIDArg)
		os.Exit(1)
	}
	sourcePaperID, err := strconv.ParseInt(paperIDStr, 10, 64)
	if err != nil {
		panic(err)
	}

	filename := fmt.Sprintf("%v.txt", paperIDStr)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	wrBuf := bufio.NewWriter(f)
	defer wrBuf.Flush()
	textEncoder := bfs.NewTextEncoder(wrBuf)
	defer textEncoder.Close()

	srcIndex, err := bfs.OpenIndex("./mag_src_hash", true, *fdLimit)
	if err != nil {
		panic(err)
	}

	dstIndex, err := bfs.OpenIndex("./mag_dst_hash", false, *fdLimit)
	if err != nil {
		panic(err)
	}

	var traverser *bfs.BFSTraverser
	if *bloom {
		traverser = bfs.NewBFSTraverserBloom(srcIndex, dstIndex)
	} else {
		traverser = bfs.NewBFSTraverserMap(srcIndex, dstIndex)
	}

	//fmt.Printf("Starting from node %d and BFSing for %d levels (followOut: %v, followIn %v)...\n", sourcePaperID, *levels, *followOut, *followIn)

	numPapers, numEdges, err := traverser.BFS([]int64{sourcePaperID}, *levels, *followIn, *followOut, func(srcID int64, dstID int64) {
		//fmt.Printf("Found %d edges at level %d\n", len(edges), level)
		textEncoder.Encode(bfs.Edge{SrcID: srcID, DstID: dstID})

	})

	report := ResultsReport{
		SourcePaperID:    sourcePaperID,
		NumberBFSLevels:  *levels,
		FollowedOutLinks: *followOut,
		FollowedInLinks:  *followIn,

		OutputEdgeList: filename,
		NumFoundNodes:  numPapers,
		NumFoundEdges:  numEdges,
	}

	jsonText, err := json.Marshal(report)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
	}
	fmt.Println(string(jsonText))
}

type ResultsReport struct {
	SourcePaperID    int64 `json:"sourcePaperID"`
	NumberBFSLevels  int   `json:"numberBFSLevels"`
	FollowedOutLinks bool  `json:"followedOutLinks"`
	FollowedInLinks  bool  `json:"followedInLinks"`

	OutputEdgeList string `json:"outputEdgeList"`
	NumFoundEdges  int64  `json:"numFoundEdges"`
	NumFoundNodes  int64  `json:"numFoundNodes"`
}
