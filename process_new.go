package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"./bfs"
)

const (
	MaxID         = 3000000000
	SrcHashFolder = "mag_src_hash"
	DstHashFolder = "mag_dst_hash"
)

func HashFilename(i int64) string {
	return fmt.Sprintf("%04d.txt", i)
}

func CloseAllFiles(files []*os.File) {
	for _, f := range files {
		f.Close()
	}
}

func FlushAllBuffers(buffers []*bufio.Writer) {
	for _, b := range buffers {
		b.Flush()
	}
}

func main() {
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var paperReferencesFile = flag.String("paperReferencesFile", "PaperReferences.txt", "Source MAG file path")
	var destHash = flag.Bool("destHash", false, "Hash based on destination id (defaults to source id)")
	var compress = flag.Bool("compress", true, "Gzip hash buckets")
	var buckets = flag.Int("buckets", 3000, "How many file buckets to hash to")
	flag.Parse()
	if *destHash {
		fmt.Printf("Reading %s into %v buckets by dst id\n", *paperReferencesFile, *buckets)
	} else {
		fmt.Printf("Reading %s into %v buckets by src id\n", *paperReferencesFile, *buckets)
	}
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

	// ================= Make hash folders =================

	if !*destHash {
		if _, err := os.Stat(SrcHashFolder); os.IsNotExist(err) {
			os.Mkdir(SrcHashFolder, os.ModePerm)
		} else {
			panic(fmt.Sprintf("Folder %v exists", SrcHashFolder))
		}
	} else {
		if _, err := os.Stat(DstHashFolder); os.IsNotExist(err) {
			os.Mkdir(DstHashFolder, os.ModePerm)
		} else {
			panic(fmt.Sprintf("Folder %v exists", DstHashFolder))
		}
	}

	var index *bfs.EdgeIndex
	var err error
	if !*destHash {
		index, err = bfs.OpenIndex(SrcHashFolder, true, 50)
		if err != nil {
			panic(err)
		}
	} else {
		index, err = bfs.OpenIndex(DstHashFolder, false, 50)
		if err != nil {
			panic(err)
		}
	}

	err = index.PopulateIndex(*paperReferencesFile, *buckets, *compress)
	if err != nil {
		panic(err)
	}

}
