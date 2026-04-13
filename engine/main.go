package main

import (
	"bufio"
	"container/heap"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

// -------------------- CONFIG --------------------

type Config struct {
	NumSlots    int
	SlotSeconds int64
	MaxTop      int
	CMSDepth    int
	CMSWidth    int
}

func DefaultConfig() Config {
	return Config{
		NumSlots:    1000,
		SlotSeconds: 5,
		MaxTop:      100,
		CMSDepth:    4,
		CMSWidth:    2000,
	}
}

// -------------------- TYPES --------------------

type Item struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
	index int
}

type Bucket struct {
	timestamp int64
	cms       *CMS
	topMap    map[string]*Item
	topHeap   MinHeap
}

type Query struct {
	K      int    `json:"k"`
	Since  int64  `json:"since"`
	Filter string `json:"filter"`
}

// -------------------- ENGINE --------------------

type Engine struct {
	now    func() int64
	config Config
	slots  []Bucket
}

func NewEngine(cfg Config, now func() int64) *Engine {
	return &Engine{
		now:    now,
		config: cfg,
		slots:  make([]Bucket, cfg.NumSlots),
	}
}

// -------------------- INGEST --------------------

func (e *Engine) Ingest(tag string) {
	now := e.now()

	slotTime := now / e.config.SlotSeconds
	idx := int(slotTime % int64(e.config.NumSlots))

	slot := &e.slots[idx]

	// rotate slot
	if slot.timestamp != slotTime {
		slot.timestamp = slotTime
		slot.cms = NewCMS(e.config.CMSDepth, e.config.CMSWidth)
		slot.topMap = make(map[string]*Item)
		slot.topHeap = MinHeap{}
	}

	// update CMS
	slot.cms.Add(tag)
	count := slot.cms.Estimate(tag)

	// case 1: already exists
	if item, ok := slot.topMap[tag]; ok {
		item.Count = count
		heap.Fix(&slot.topHeap, item.index)
		return
	}

	// case 2: space available
	if len(slot.topHeap) < e.config.MaxTop {
		item := &Item{Tag: tag, Count: count}
		heap.Push(&slot.topHeap, item)
		slot.topMap[tag] = item
		return
	}

	// case 3: replace min
	minItem := slot.topHeap[0]
	if count > minItem.Count {
		removed := heap.Pop(&slot.topHeap).(*Item)
		delete(slot.topMap, removed.Tag)

		item := &Item{Tag: tag, Count: count}
		heap.Push(&slot.topHeap, item)
		slot.topMap[tag] = item
	}
}

// -------------------- QUERY --------------------

func (e *Engine) Query(q Query) []Item {
	now := e.now()
	cutoff := now - q.Since
	cutoffSlot := cutoff / e.config.SlotSeconds

	merged := NewCMS(e.config.CMSDepth, e.config.CMSWidth)
	candidates := make(map[string]struct{})

	for i := 0; i < e.config.NumSlots; i++ {
		slot := e.slots[i]

		if slot.cms == nil || slot.topMap == nil {
			continue
		}

		if slot.timestamp < cutoffSlot {
			continue
		}

		merged.Merge(slot.cms)

		for tag := range slot.topMap {
			candidates[tag] = struct{}{}
		}
	}

	var results []Item

	for tag := range candidates {
		if q.Filter != "" && !strings.Contains(tag, q.Filter) {
			continue
		}

		count := merged.Estimate(tag)

		results = append(results, Item{
			Tag:   tag,
			Count: count,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Count > results[j].Count
	})

	if len(results) > q.K {
		results = results[:q.K]
	}

	return results
}

// -------------------- SOCKET SERVER --------------------

func startSocketServer(engine *Engine) {
	socketPath := "/tmp/hashbang.sock"
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	fmt.Println("socket listening at", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, engine)
	}
}

func handleConnection(conn net.Conn, engine *Engine) {
	defer conn.Close()

	var query Query

	if err := json.NewDecoder(conn).Decode(&query); err != nil {
		return
	}

	result := engine.Query(query)
	json.NewEncoder(conn).Encode(result)
}

// -------------------- MAIN --------------------

func main() {
	cfg := DefaultConfig()

	engine := NewEngine(cfg, func() int64 {
		return time.Now().Unix()
	})

	go startSocketServer(engine)

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		tag := scanner.Text()
		engine.Ingest(tag)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("error:", err)
	}
}