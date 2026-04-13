package main

import (
	"hash/fnv"
	"strconv"
)

type CMS struct {
	depth int
	width int
	table [][]int
}

func NewCMS(depth, width int) *CMS {
	table := make([][]int, depth)
	for i := range table {
		table[i] = make([]int, width)
	}
	return &CMS{
		depth: depth,
		width: width,
		table: table,
	}
}

func (c *CMS) hash(item string, i int) int {
	h := fnv.New32a()
	h.Write([]byte(item))
	h.Write([]byte(strconv.Itoa(i)))
	return int(h.Sum32()) % c.width
}

func (c *CMS) Add(item string) {
	for i := 0; i < c.depth; i++ {
		idx := c.hash(item, i)
		c.table[i][idx]++
	}
}

func (c *CMS) Estimate(item string) int {
	min := int(^uint(0) >> 1) // max int

	for i := 0; i < c.depth; i++ {
		idx := c.hash(item, i)
		if c.table[i][idx] < min {
			min = c.table[i][idx]
		}
	}

	return min
}

func (c *CMS) Merge(other *CMS) {
	for i := 0; i < c.depth; i++ {
		for j := 0; j < c.width; j++ {
			c.table[i][j] += other.table[i][j]
		}
	}
}