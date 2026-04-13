package main


type MinHeap []*Item

func (h MinHeap) Len() int { return len(h) }

func (h MinHeap) Less(i, j int) bool {
	return h[i].Count < h[j].Count
}

func (h MinHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *MinHeap) Push(x interface{}) {
	item := x.(*Item)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *MinHeap) Pop() interface{} {
	old := *h
	n := len(old)

	item := old[n-1]
	item.index = -1

	*h = old[0 : n-1]
	return item
}