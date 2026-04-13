package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

type Query struct {
	K      int
	Since  int64
	Filter string
}

type Item struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
	Index int    `json:"index"`
}

// ✅ Allowed windows
var allowedWindows = map[string]int64{
	"5m":  300,
	"10m": 600,
	"30m": 1800,
	"1h":  3600,
}

// ✅ Pretty print allowed values
func printAllowedWindows() {
	fmt.Println("invalid since value. Allowed values:")
	fmt.Println("5m  (5 minutes)")
	fmt.Println("10m (10 minutes)")
	fmt.Println("30m (30 minutes)")
	fmt.Println("1h  (1 hour)")
}

// ✅ Parse human-friendly input
func parseSince(input string) (int64, error) {
	val, ok := allowedWindows[input]
	if !ok {
		return 0, fmt.Errorf("invalid since")
	}
	return val, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: cli <k> [since: 5m|10m|30m|1h] [filter]")
		return
	}

	// ✅ Parse K
	k, err := strconv.Atoi(os.Args[1])
	if err != nil || k <= 0 {
		fmt.Println("invalid k")
		return
	}

	// ✅ Default window = 5 minutes
	var since int64 = 300

	if len(os.Args) > 2 {
		since, err = parseSince(os.Args[2])
		if err != nil {
			printAllowedWindows()
			return
		}
	}

	// ✅ Optional filter
	filter := ""
	if len(os.Args) > 3 {
		filter = os.Args[3]
	}

	// ✅ Connect to socket
	conn, err := net.Dial("unix", "/tmp/hashbang.sock")
	if err != nil {
		fmt.Println("connection error:", err)
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	query := Query{
		K:      k,
		Since:  since,
		Filter: filter,
	}

	// ✅ Send query
	if err := json.NewEncoder(conn).Encode(query); err != nil {
		fmt.Println("encode error:", err)
		return
	}

	// ✅ Receive response
	var result []Item
	if err := json.NewDecoder(conn).Decode(&result); err != nil {
		fmt.Println("decode error:", err)
		return
	}

	// ✅ Print results
	if len(result) == 0 {
		fmt.Println("No results")
		return
	}

	fmt.Printf("Top %d results (last %ds):\n", k, since)
	for _, item := range result {
		fmt.Printf("%s → %d\n", item.Tag, item.Count)
	}
}