package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	pb "gopkg.in/cheggaaa/pb.v1"
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
	var paperReferencesFile = flag.String("paperReferencesFile", "PaperReferences.txt", "Source MAG file path")
	var destHash = flag.Bool("destHash", false, "Hash based on destination id (defaults to source id)")
	var buckets = flag.Int("buckets", 3000, "How many file buckets to hash to")
	flag.Parse()
	if *destHash {
		fmt.Printf("Reading %s into %v buckets by dst id\n", *paperReferencesFile, *buckets)
	} else {
		fmt.Printf("Reading %s into %v buckets by src id\n", *paperReferencesFile, *buckets)
	}

	// ================= Open source file =================
	file, err := os.Open(*paperReferencesFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

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

	// ================= Make hash files =================

	srcFiles := make([]*os.File, *buckets)
	dstFiles := make([]*os.File, *buckets)
	if !*destHash {
		for i := int64(0); i < int64(len(srcFiles)); i++ {
			f, err := os.OpenFile(path.Join(SrcHashFolder, HashFilename(i)), os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(err)
			}
			srcFiles[i] = f
		}
		defer CloseAllFiles(srcFiles)
	} else {

		for i := int64(0); i < int64(len(dstFiles)); i++ {
			f, err := os.OpenFile(path.Join(DstHashFolder, HashFilename(i)), os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(err)
			}
			dstFiles[i] = f
		}
		defer CloseAllFiles(dstFiles)
	}

	// ================= Make hash buffers =================
	srcBuffers := make([]*bufio.Writer, len(srcFiles))
	dstBuffers := make([]*bufio.Writer, len(dstFiles))
	if !*destHash {
		for i, file := range srcFiles {
			srcBuffers[i] = bufio.NewWriter(file)
		}
		defer FlushAllBuffers(srcBuffers)
	} else {

		for i, file := range dstFiles {
			dstBuffers[i] = bufio.NewWriter(file)
		}
		defer FlushAllBuffers(dstBuffers)
	}

	// ================= Distribute files =================
	bar := pb.StartNew(1369744602) // Number of lines in file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// =============== Parse IDs ===============
		ids := strings.Split(scanner.Text(), "\t")
		if !*destHash {
			srcID, err := strconv.ParseInt(ids[0], 10, 64)
			if err != nil {
				panic("Invalid src_id")
			}
			// =============== Write to src hash ===============
			srcBucketID := srcID % int64(len(srcBuffers))
			srcBuffers[srcBucketID].Write(scanner.Bytes())
			srcBuffers[srcBucketID].Write([]byte("\n"))
		} else {
			dstID, err := strconv.ParseInt(ids[1], 10, 64)
			if err != nil {
				panic("Invalid dst_id")
			}

			// =============== Write to dst hash ===============
			dstBucketID := dstID % int64(len(dstBuffers))
			dstBuffers[dstBucketID].Write(scanner.Bytes())
			dstBuffers[dstBucketID].Write([]byte("\n"))
		}

		bar.Increment()
	}
	bar.FinishPrint("Finished building hash tables")
}
