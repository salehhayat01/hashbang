package main

import (
	"container/heap"
	"fmt"
	"testing"
)

// -------------------- HELPERS --------------------

func newTestEngine(start int64) (*Engine, *int64) {
	t := start

	engine := NewEngine(Config{
		NumSlots:    10,
		SlotSeconds: 5,
		MaxTop:      5,
		CMSDepth:    4,
		CMSWidth:    50,
	}, func() int64 {
		return t
	})

	return engine, &t
}

// -------------------- MASTER TEST --------------------

func TestEngine_Extreme_Isolated(t *testing.T) {

	t.Run("Basic Ingest", func(t *testing.T) {
		engine, _ := newTestEngine(1000)

		engine.Ingest("a")
		engine.Ingest("a")
		engine.Ingest("b")

		res := engine.Query(Query{K: 2, Since: 60})
		if len(res) == 0 || res[0].Tag != "a" {
			t.Fatalf("basic ingest failed")
		}
	})

	t.Run("Rotation + Aggregation", func(t *testing.T) {
		engine, fakeTime := newTestEngine(1000)

		engine.Ingest("a")
		engine.Ingest("a")

		*fakeTime += 5
		engine.Ingest("a")
		engine.Ingest("b")

		*fakeTime += 5
		engine.Ingest("a")
		engine.Ingest("c")

		res := engine.Query(Query{K: 3, Since: 60})

		expected := map[string]int{"a": 4, "b": 1, "c": 1}

		for _, r := range res {
			if exp, ok := expected[r.Tag]; ok {
				if r.Count < exp {
					t.Fatalf("aggregation failed for %s", r.Tag)
				}
			}
		}
	})

	t.Run("Boundary Condition", func(t *testing.T) {
		engine, fakeTime := newTestEngine(1000)

		engine.Ingest("a")
		*fakeTime += 5

		res := engine.Query(Query{K: 10, Since: 5})

		if len(res) == 0 {
			t.Fatalf("boundary condition dropped valid data")
		}
	})

	t.Run("Slot Overwrite", func(t *testing.T) {
		engine, fakeTime := newTestEngine(1000)

		engine.Ingest("old")

		*fakeTime += int64(engine.config.NumSlots) * engine.config.SlotSeconds

		engine.Ingest("new")

		res := engine.Query(Query{K: 10, Since: 10})

		for _, r := range res {
			if r.Tag == "old" {
				t.Fatalf("old data leaked after overwrite")
			}
		}
	})

	t.Run("Heap Integrity", func(t *testing.T) {
		engine, _ := newTestEngine(1000)

		for i := 0; i < 10; i++ {
			tag := string(rune('a' + i))
			for j := 0; j < i+1; j++ {
				engine.Ingest(tag)
			}
		}

		for _, slot := range engine.slots {
			for i, item := range slot.topHeap {
				if item.index != i {
					t.Fatalf("heap index corrupted")
				}
			}
		}
	})

	t.Run("CMS Merge Drift", func(t *testing.T) {
		engine, fakeTime := newTestEngine(1000)

		for i := 0; i < 50; i++ {
			engine.Ingest("x")
		}

		*fakeTime += 5

		for i := 0; i < 50; i++ {
			engine.Ingest("x")
		}

		res := engine.Query(Query{K: 1, Since: 60})

		if len(res) == 0 || res[0].Count < 100 {
			t.Fatalf("CMS merge drift detected")
		}
	})

	t.Run("Candidate Loss", func(t *testing.T) {
		engine, _ := newTestEngine(1000)

		for i := 0; i < 100; i++ {
			engine.Ingest(string(rune('a' + (i % 26))))
		}

		for i := 0; i < 50; i++ {
			engine.Ingest("z")
		}

		res := engine.Query(Query{K: 5, Since: 60})

		found := false
		for _, r := range res {
			if r.Tag == "z" {
				found = true
			}
		}

		if !found {
			t.Fatalf("candidate loss detected")
		}
	})

	t.Run("Skew Distribution", func(t *testing.T) {
		engine, fakeTime := newTestEngine(1000)

		// ensure fresh slot
		*fakeTime += 5

		for i := 0; i < 10000; i++ {
			engine.Ingest("hot")
		}

		for i := 0; i < 10000; i++ {
			engine.Ingest(string(rune('a' + i%26)))
		}

		res := engine.Query(Query{K: 1, Since: 60})

		if len(res) == 0 || res[0].Tag != "hot" {
			t.Fatalf("skew distribution failed: %+v", res)
		}
	})

	t.Run("Heap Sanity", func(t *testing.T) {
		h := &MinHeap{}

		heap.Push(h, &Item{Tag: "a", Count: 5})
		heap.Push(h, &Item{Tag: "b", Count: 2})
		heap.Push(h, &Item{Tag: "c", Count: 10})

		min := heap.Pop(h).(*Item)

		if min.Tag != "b" {
			t.Fatalf("heap min property broken")
		}
	})
}
// -------------------- CONCURRENCY TEST --------------------

func TestEngine_Concurrent(t *testing.T) {
	engine, _ := newTestEngine(1000)

	done := make(chan bool)

	go func() {
		for i := 0; i < 20000; i++ {
			engine.Ingest("a")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 20000; i++ {
			engine.Query(Query{K: 5, Since: 60})
		}
		done <- true
	}()

	<-done
	<-done
}

func TestEngine_GlobalLossWithWideQuery(t *testing.T) {
	engine, fakeTime := newTestEngine(1000)

	numSlots := 5

	for s := 0; s < numSlots; s++ {

		// move to new slot
		*fakeTime += engine.config.SlotSeconds

		// hot appears in EVERY slot
		for i := 0; i < 100; i++ {
			engine.Ingest("hot")
		}

		// 5 competitors per slot (each slightly stronger)
		for c := 0; c < engine.config.MaxTop; c++ {

			// unique per slot → increases global diversity
			tag := fmt.Sprintf("comp_%d_%d", s, c)

			for i := 0; i < 101; i++ {
				engine.Ingest(tag)
			}
		}
	}

	// query LARGE K
	res := engine.Query(Query{
		K:     100,
		Since: 60,
	})

	t.Logf("results: %+v", res)

	// check if hot exists
	foundHot := false
	for _, r := range res {
		if r.Tag == "hot" {
			foundHot = true
		}
	}

	// 🔥 THIS SHOULD FAIL (design limitation)
	if foundHot {
		t.Fatalf("hot should not appear due to know issue of borderline misses")
	}
}