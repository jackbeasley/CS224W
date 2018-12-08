package bfs

import (
	"encoding/binary"

	boom "github.com/tylertreat/BoomFilters"
)

type BFSTraverser struct {
	srcIndex *EdgeIndex
	dstIndex *EdgeIndex
	store    NodeStore
}

type NodeStore interface {
	HasSeen(nodeID int64) bool
	Mark(nodeID int64)
	Clear()
}

type BloomNodeStore struct {
	inverseBloom *boom.InverseBloomFilter
}

func NewBloomNodeStore(size int) NodeStore {
	return &BloomNodeStore{
		inverseBloom: boom.NewInverseBloomFilter(uint(size)),
	}
}

func (bns *BloomNodeStore) HasSeen(nodeID int64) bool {
	bs := make([]byte, 8)
	binary.PutVarint(bs, nodeID)
	return bns.inverseBloom.Test(bs)
}

func (mns *BloomNodeStore) Mark(nodeID int64) {
	bs := make([]byte, 8)
	binary.PutVarint(bs, nodeID)
	mns.inverseBloom.Add(bs)
}

func (mns *BloomNodeStore) Clear() {
	size := mns.inverseBloom.Capacity()
	mns.inverseBloom = nil
	mns.inverseBloom = boom.NewInverseBloomFilter(size)
}

type MapNodeStore struct {
	nodes map[int64]struct{}
}

func NewMapNodeStore(size int) NodeStore {
	return &MapNodeStore{
		nodes: make(map[int64]struct{}, size),
	}
}

func (mns *MapNodeStore) HasSeen(nodeID int64) bool {
	_, ok := mns.nodes[nodeID]
	return ok
}

func (mns *MapNodeStore) Mark(nodeID int64) {
	mns.nodes[nodeID] = struct{}{}
}

func (mns *MapNodeStore) Clear() {
	size := len(mns.nodes)
	mns.nodes = nil
	mns.nodes = make(map[int64]struct{}, size)
}

func NewBFSTraverserBloom(srcIndex, dstIndex *EdgeIndex) *BFSTraverser {
	return NewBFSTraverser(srcIndex, dstIndex, NewBloomNodeStore(100000000))
}

func NewBFSTraverserMap(srcIndex, dstIndex *EdgeIndex) *BFSTraverser {
	return NewBFSTraverser(srcIndex, dstIndex, NewMapNodeStore(100000))
}

func NewBFSTraverser(srcIndex, dstIndex *EdgeIndex, store NodeStore) *BFSTraverser {
	return &BFSTraverser{
		srcIndex: srcIndex,
		dstIndex: dstIndex,
		store:    store,
	}
}

func (bt *BFSTraverser) BFS(initPapers []int64, levels int, followIn, followOut bool, callback func([]Edge)) (int64, int64, error) {
	defer bt.store.Clear()
	currentPapers := make(map[int64]bool, 10000) // Will quickly grow to be quite large
	for _, paperID := range initPapers {
		currentPapers[paperID] = true
		bt.store.Mark(paperID)
	}

	numPapers := int64(len(initPapers))
	numEdges := int64(0)
	for i := 0; i < levels; i++ {
		var outRefsAtLevel []Edge
		var err error
		if followOut {
			outRefsAtLevel, err = bt.srcIndex.GetReferencedEdges(currentPapers, true, false)
			if err != nil {
				return 0, 0, err
			}
		}
		var inRefsAtLevel []Edge
		if followIn {
			inRefsAtLevel, err = bt.dstIndex.GetReferencedEdges(currentPapers, false, true)
			if err != nil {
				return 0, 0, err
			}
		}
		refsFound := make([]Edge, 0, len(outRefsAtLevel)+len(inRefsAtLevel))
		refsFound = append(refsFound, outRefsAtLevel...)
		refsFound = append(refsFound, inRefsAtLevel...)
		currentPapers = nil
		currentPapers = make(map[int64]bool, 10000)
		for _, edge := range refsFound {
			numEdges++
			if !bt.store.HasSeen(edge.SrcID) {
				numPapers++
				currentPapers[edge.SrcID] = true
				bt.store.Mark(edge.SrcID)
			}
			if !bt.store.HasSeen(edge.DstID) {
				numPapers++
				currentPapers[edge.DstID] = true
				bt.store.Mark(edge.DstID)
			}
		}
		callback(refsFound)
	}
	return numPapers, numEdges, nil
}
