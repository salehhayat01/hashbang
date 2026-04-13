package main

import (
	"container/heap"
	"testing"
)

func TestMinHeap(t *testing.T) {
	h := &MinHeap{}

	heap.Push(h, &Item{Tag: "a", Count: 5})
	heap.Push(h, &Item{Tag: "b", Count: 2})
	heap.Push(h, &Item{Tag: "c", Count: 10})

	min := heap.Pop(h).(*Item)

	if min.Tag != "b" {
		t.Errorf("expected b, got %s", min.Tag)
	}
}
func TestHeapFix(t *testing.T) {
	h := &MinHeap{}

	a := &Item{Tag: "a", Count: 5}
	b := &Item{Tag: "b", Count: 2}

	heap.Push(h, a)
	heap.Push(h, b)

	// increase b
	b.Count = 20
	heap.Fix(h, b.index)

	min := (*h)[0]

	if min.Tag != "a" {
		t.Errorf("expected a as min after fix, got %s", min.Tag)
	}
}