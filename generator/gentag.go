package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	trendingSets := [][]string{
		{"hot", "ai", "crypto"},
		{"news", "cloud", "server"},
		{"golang", "api", "backend"},
	}

	longTail := []string{
		"apple", "banana", "car", "tree",
		"river", "phone", "book", "music",
	}

	slot := 0

	for {
		// rotate trending every few seconds
		if rand.Intn(100) < 2 {
			slot = (slot + 1) % len(trendingSets)
		}

		var word string

		switch rand.Intn(10) {
		case 0, 1, 2:
			word = trendingSets[slot][rand.Intn(len(trendingSets[slot]))]

		default:
			word = longTail[rand.Intn(len(longTail))]
		}

		fmt.Println(word)
		time.Sleep(10 * time.Millisecond)
	}
}