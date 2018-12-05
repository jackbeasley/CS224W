package bfs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	pb "gopkg.in/cheggaaa/pb.v1"
)

type EdgeIndex struct {
	indexSrc bool
	path     string
	numFiles int64
}

func OpenIndex(path string, indexSrc bool) (*EdgeIndex, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(absPath, "*"))
	if err != nil {
		return nil, err
	}
	fmt.Println(files)

	return &EdgeIndex{
		indexSrc: indexSrc,
		path:     absPath,
		numFiles: int64(len(files)),
	}, nil
}

func (ei EdgeIndex) nodeIDToFilename(nodeID int64) string {
	return fmt.Sprintf("%d.ind", nodeID%ei.numFiles)
}

func (ei EdgeIndex) GetReaderForNode(nodeID int64) (*os.File, error) {
	file, err := os.Open(filepath.Join(ei.path, ei.nodeIDToFilename(nodeID)))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (ei EdgeIndex) GetWriterForNode(nodeID int64) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(ei.path, ei.nodeIDToFilename(nodeID)), os.O_WRONLY, 0755)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (ei EdgeIndex) getAllWriters() ([]*os.File, error) {
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

func (ei EdgeIndex) PopulateIndex(sourceFile string, numFiles int) error {
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
		encoders[i] = NewBinaryEncoder(bw, false)
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
			encoders[edge.SrcID%ei.numFiles].Encode(edge)
		} else {
			encoders[edge.DstID%ei.numFiles].Encode(edge)
		}
		bar.Increment()
	}
	bar.FinishPrint("Finished building hash tables")

	return nil

}
