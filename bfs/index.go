package bfs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	pb "gopkg.in/cheggaaa/pb.v1"
)

type EdgeIndex struct {
	indexSrc    bool
	path        string
	numFiles    int64
	parallelism int
}

func OpenIndex(path string, indexSrc bool, parallelism int) (*EdgeIndex, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(absPath, "*"))
	if err != nil {
		return nil, err
	}

	return &EdgeIndex{
		indexSrc:    indexSrc,
		path:        absPath,
		numFiles:    int64(len(files)),
		parallelism: parallelism,
	}, nil
}

func (ei *EdgeIndex) nodeIDToFileID(nodeID int64) int {
	return int(nodeID % ei.numFiles)
}

func (ei *EdgeIndex) fileIDToFilename(fileID int) string {
	return fmt.Sprintf("%d.ind", fileID)
}

func (ei *EdgeIndex) nodeIDToFilename(nodeID int64) string {
	return ei.fileIDToFilename(ei.nodeIDToFileID(nodeID))
}

func (ei *EdgeIndex) GetReaderForFileID(fileID int) (*os.File, error) {
	file, err := os.Open(filepath.Join(ei.path, ei.fileIDToFilename(fileID)))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (ei *EdgeIndex) matchEdgesInFile(fileID int, paperIDs map[int64]bool, matchSrc, matchDst bool) ([]Edge, error) {
	reader, err := ei.GetReaderForFileID(fileID)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	readBuf := bufio.NewReader(reader)

	d, err := NewBinaryDecoder(readBuf, true)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	edges := make([]Edge, 0, 100000)
	for edge, err := d.Decode(); err != io.EOF; edge, err = d.Decode() {
		if err != nil {
			return nil, err
		}
		if (matchSrc && paperIDs[edge.SrcID]) || (matchDst && paperIDs[edge.DstID]) {
			edges = append(edges, edge)
		}
	}
	return edges, nil
}

func (ei *EdgeIndex) GetReferencedEdges(paperIDs map[int64]bool, matchSrc, matchDst bool) ([]Edge, error) {
	fileIDsToPapers := make(map[int]map[int64]bool, len(paperIDs))
	for paperID := range paperIDs {
		if fileIDsToPapers[ei.nodeIDToFileID(paperID)] == nil {
			fileIDsToPapers[ei.nodeIDToFileID(paperID)] = make(map[int64]bool, 5)
		}
		fileIDsToPapers[ei.nodeIDToFileID(paperID)][paperID] = true
	}
	results := make([][]Edge, len(fileIDsToPapers))
	errors := make([]error, len(fileIDsToPapers))
	resultsIndex := 0

	var wg sync.WaitGroup
	// Use a channel like a semaphore so we can limit parallelism,
	// primarily limiting file descriptors
	parallelismSemaphoreChan := make(chan struct{}, ei.parallelism)

	for fileID, paperIDsInFile := range fileIDsToPapers {
		wg.Add(1)
		go func(fileID int, ids map[int64]bool, resultsIndex int) {
			parallelismSemaphoreChan <- struct{}{}
			defer wg.Done()
			edges, err := ei.matchEdgesInFile(fileID, ids, matchSrc, matchDst)
			if err != nil {
				errors[resultsIndex] = err
			} else {
				results[resultsIndex] = edges
			}
			<-parallelismSemaphoreChan
		}(fileID, paperIDsInFile, resultsIndex)
		resultsIndex++
	}
	wg.Wait()
	for _, err := range errors {
		// TODO: collect errors
		if err != nil {
			return nil, err
		}
	}
	edges := make([]Edge, 0, len(results)*100)
	for _, resultList := range results {
		edges = append(edges, resultList...)
	}
	return edges, nil
}

func (ei *EdgeIndex) GetWriterForNode(nodeID int64) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(ei.path, ei.nodeIDToFilename(nodeID)), os.O_WRONLY, 0755)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (ei *EdgeIndex) getAllWriters() ([]*os.File, error) {
	files := make([]*os.File, ei.numFiles)
	for i := int64(0); i < ei.numFiles; i++ {
		file, err := os.OpenFile(filepath.Join(ei.path, ei.nodeIDToFilename(i)), os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return nil, err
		}
		files[i] = file
	}
	return files, nil
}

func (ei *EdgeIndex) PopulateIndex(sourceFile string, numFiles int, compress bool) error {
	ei.numFiles = int64(numFiles)
	writers, err := ei.getAllWriters()
	if err != nil {
		return err
	}
	defer func() {
		for _, wr := range writers {
			wr.Close()
		}
	}()
	buffers := make([]*bufio.Writer, len(writers))
	for i, wr := range writers {
		buffers[i] = bufio.NewWriter(wr)
	}
	defer func() {
		for _, bw := range buffers {
			bw.Flush()
		}
	}()

	encoders := make([]*BinaryEncoder, len(writers))
	for i, bw := range buffers {
		encoders[i] = NewBinaryEncoder(bw, compress)
	}
	defer func() {
		for _, enc := range encoders {
			enc.Close()
		}
	}()
	file, err := os.Open(sourceFile)
	if err != nil {
		return err
	}

	d := NewTextDecoder(file)
	bar := pb.StartNew(1369744602) // Number of lines in file
	for edge, err := d.Decode(); err != io.EOF; edge, err = d.Decode() {
		if err != nil {
			return err
		}
		if ei.indexSrc {
			encoders[ei.nodeIDToFileID(edge.SrcID)].Encode(edge)
		} else {
			encoders[ei.nodeIDToFileID(edge.DstID)].Encode(edge)
		}
		bar.Increment()
	}
	bar.FinishPrint("Finished building hash tables")

	return nil

}
