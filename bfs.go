package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"strconv"
	"sync"
)

const (
	srcHashDir    = "mag_src_hash"
	dstHashDir    = "mag_dst_hash"
	errPaperIDArg = "You must provide an integer paper ID as an argument"
)

func main() {
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")

	var followOut = flag.Bool("out", false, "Follow out links (i.e. cited papers)")
	var followIn = flag.Bool("in", false, "Follow in links (i.e. paper who cite)")
	var levels = flag.Int("levels", 1, "How many levels to BFS out")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	paperIDStr := flag.Arg(0)
	if paperIDStr == "" {
		fmt.Println(errPaperIDArg)
		os.Exit(1)
	}
	sourcePaperID, err := strconv.ParseInt(paperIDStr, 10, 64)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Starting from %v and searcher for %v levels (follow out: %v, follow in: %v)\n", sourcePaperID, *levels, *followOut, *followIn)

	papersToReferences, seenPapers := Bfs(sourcePaperID, *followIn, *followOut, *levels)

	numEdges := 0
	for _, references := range papersToReferences {
		numEdges += len(references)
	}
	filename := fmt.Sprintf("%v.txt", paperIDStr)
	fmt.Printf("Found %v nodes, %v edges and output edge list to %v", seenPapers, numEdges, filename)

	writeEdgeList(filename, papersToReferences)
}

func writeEdgeList(filename string, graph map[int64]map[int64]bool) {
	if _, err := os.Stat(filename); err == nil || !os.IsNotExist(err) {
		os.Remove(filename)
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	wrBuf := bufio.NewWriter(f)
	defer wrBuf.Flush()

	for srcID, references := range graph {
		for dstID := range references {
			wrBuf.Write([]byte(fmt.Sprintf("%v\t%v\n", srcID, dstID)))
		}
	}

}

func hashFilename(i int64) string {
	return fmt.Sprintf("%04d.txt", i)
}

func GetSrcFilenameForPaperID(paperID int64) string {
	return path.Join(srcHashDir, hashFilename(paperID%3000))
}

func GetDstFilenameForPaperID(paperID int64) string {
	return path.Join(dstHashDir, hashFilename(paperID%3000))
}

func Bfs(initPaperID int64, followIn, followOut bool, levels int) (map[int64]map[int64]bool, int) {
	papersToReferences := make(map[int64]map[int64]bool, levels*10)

	seenPapers := make(map[int64]bool, 5*levels*levels) // Like a set
	seenPapers[initPaperID] = true
	currentPapers := make(map[int64]bool, 1) // Like a set
	currentPapers[initPaperID] = true

	for i := 0; i < levels; i++ {
		paperRefsAtLevel := GetReferences(currentPapers, followIn, followOut)
		for paperID, references := range paperRefsAtLevel {
			if _, ok := papersToReferences[paperID]; !ok {
				papersToReferences[paperID] = references
			} else {
				// Merge edges found in different files, happens when finding in
				// links or when finding both in and out at the same time
				// fmt.Printf("Merging edges found in different files for paper: %v\n", paperID)
				for refID := range references {
					papersToReferences[paperID][refID] = true
				}
			}
		}
		currentPapers = nil
		currentPapers = make(map[int64]bool, len(paperRefsAtLevel)*5) // Like a set
		for paperID, references := range paperRefsAtLevel {
			if _, ok := seenPapers[paperID]; !ok {
				seenPapers[paperID] = true
				currentPapers[paperID] = true
			}

			for refID := range references {
				if _, ok := seenPapers[refID]; !ok {
					seenPapers[refID] = true
					currentPapers[refID] = true
				}
			}
		}
	}
	return papersToReferences, len(seenPapers)
}

// This function allows us to do allocation free parsing of each lines
func parseBytes(bts []byte) int64 {
	var x int64
	for _, c := range bts {
		x = x*10 + int64(c-'0')
	}
	return x
}

// Match src if dst not true, dst if dst is true
func getRefsInFile(paperIDs map[int64]bool, refFile string, dst bool) map[int64]map[int64]bool {
	f, err := os.Open(refFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	bufRead := bufio.NewReaderSize(f, 1000000)

	scanner := bufio.NewScanner(bufRead)

	papersToReferences := make(map[int64]map[int64]bool, len(paperIDs)*2)
	for scanner.Scan() {
		// Keep this as Bytes to avoid the allocation associated with .Text()
		line := scanner.Bytes()
		i := bytes.Index(line, []byte("\t"))
		// Use this rather than strconv.ParseInt becuase that requires
		// converting to strings, and thus allocating strings
		srcID := parseBytes(line[:i])
		dstID := parseBytes(line[i+1:])

		if dst {
			if paperIDs[dstID] {
				if papersToReferences[srcID] == nil {
					papersToReferences[srcID] = make(map[int64]bool, 5)
				}
				papersToReferences[srcID][dstID] = true
			}
		} else {
			if paperIDs[srcID] {
				if papersToReferences[srcID] == nil {
					papersToReferences[srcID] = make(map[int64]bool, 5)
				}

				papersToReferences[srcID][dstID] = true
			}
		}
	}
	return papersToReferences
}

func GetReferences(paperIDs map[int64]bool, followIn, followOut bool) map[int64]map[int64]bool {
	var wg sync.WaitGroup
	semaphoreChan := make(chan struct{}, 100)

	inResults := make([]map[int64]map[int64]bool, len(paperIDs))
	if followIn {
		filesToInPapers := make(map[string]map[int64]bool, len(paperIDs))
		for paperID := range paperIDs {
			if filesToInPapers[GetDstFilenameForPaperID(paperID)] == nil {
				filesToInPapers[GetDstFilenameForPaperID(paperID)] = make(map[int64]bool, 5)
			}
			filesToInPapers[GetDstFilenameForPaperID(paperID)][paperID] = true
		}
		inResultNum := 0
		for file, paperIDsInFile := range filesToInPapers {
			wg.Add(1)
			go func(filename string, ids map[int64]bool, resultNum int) {
				semaphoreChan <- struct{}{}
				defer wg.Done()
				inResults[resultNum] = getRefsInFile(ids, filename, true)
				<-semaphoreChan
			}(file, paperIDsInFile, inResultNum)
			inResultNum++
		}
		wg.Wait()
	}

	outResults := make([]map[int64]map[int64]bool, len(paperIDs))
	if followOut {
		filesToOutPapers := make(map[string]map[int64]bool, len(paperIDs))
		for paperID := range paperIDs {
			if filesToOutPapers[GetSrcFilenameForPaperID(paperID)] == nil {
				filesToOutPapers[GetSrcFilenameForPaperID(paperID)] = make(map[int64]bool, 5)
			}

			filesToOutPapers[GetSrcFilenameForPaperID(paperID)][paperID] = true
		}
		outResultNum := 0
		for file, paperIDsInFile := range filesToOutPapers {
			wg.Add(1)
			go func(filename string, ids map[int64]bool, resultNum int) {
				semaphoreChan <- struct{}{}
				defer wg.Done()
				outResults[resultNum] = getRefsInFile(ids, filename, false)
				<-semaphoreChan

			}(file, paperIDsInFile, outResultNum)
			outResultNum++
		}
		wg.Wait()
	}
	allResults := append(inResults, outResults...)

	combinedResults := make(map[int64]map[int64]bool, (len(outResults)+len(inResults))*5)
	for _, result := range allResults {
		for paperID, references := range result {
			if _, ok := combinedResults[paperID]; !ok {
				combinedResults[paperID] = references
			} else {
				for refID := range references {
					combinedResults[paperID][refID] = true
				}
			}
		}
	}
	return combinedResults
}
