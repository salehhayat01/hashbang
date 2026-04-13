# hashbang

**hashbang** estimates trending tags in near real time from a continuous stream. It ingests one tag per line on standard input, maintains a **time-bucketed Count–Min Sketch (CMS)** with a **min-heap** of hot candidates per bucket, and answers **top‑K** queries over a sliding time window via a **Unix domain socket**.

---

## Features

- **Streaming ingest** — read tags from stdin (pipe from logs, generators, or other processes).
- **Approximate counts** — per-slot [Count–Min Sketch](https://en.wikipedia.org/wiki/Count%E2%80%93min_sketch) for memory-efficient frequency estimates.
- **Top candidates per time slot** — min-heap tracks up to `MaxTop` tags per bucket for query-time candidate selection.
- **Sliding-window queries** — merge sketches across buckets that overlap the requested window.
- **CLI client** — JSON query/response over `/tmp/hashbang.sock` with optional substring filter.

---

## Requirements

- [Go](https://go.dev/dl/) **1.18+** (see `go.mod`).

No third-party modules; standard library only.

---

## Repository layout

| Path | Role |
|------|------|
| [`engine/`](engine/) | Core library + **binary**: ingest loop, `Engine`, CMS, heap, Unix socket server. |
| [`cli/`](cli/) | **Binary**: connects to the socket, sends a top‑K query, prints results. |
| [`generator/`](generator/) | **Binary**: synthetic tag stream (trending bursts + long tail) for demos and load tests. |

---

## Architecture (overview)

1. **Time slots** — wall-clock time is divided into fixed-length **slots** (`SlotSeconds`, default **5s**). Each slot has its own CMS and heap; old slot data is **rotated** when a new slot index is written.
2. **Ingest** — each tag updates the current slot’s CMS; estimated frequency drives heap updates so likely hot tags stay in the candidate set.
3. **Query** — for a window `since` (seconds), the engine merges CMS tables from all slots that still overlap the window, unions **candidate tags** from those slots’ heaps, re-estimates counts from the merged sketch, optionally **filters** by substring, then returns the top **K** by count.

Counts are **approximate** (CMS property). Very rare tags may be missing from per-slot heaps even if they appear in the sketch; the design favors tracking high-volume trends.

---

## Run (end-to-end)

The engine listens on a Unix socket at **`/tmp/hashbang.sock`** and removes any existing file at startup. Run the **engine** and **CLI** in separate terminals (or background the engine).

### Terminal 1 — engine fed by the synthetic generator

```bash
go run ./generator | go run ./engine
```

### Terminal 2 — query top tags

```bash
go run ./cli <k> [since] [filter]
```

**Arguments**

| Argument | Meaning |
|----------|---------|
| `k` | Number of results (positive integer). |
| `since` | **Optional.** Window length: `5m`, `10m`, `30m`, or `1h`. Default: **`5m`**. |
| `filter` | **Optional.** If set, only tags **containing** this substring are considered. |

**Examples**

```bash
go run ./cli 10
go run ./cli 5 1h
go run ./cli 20 30m go
```

## Socket protocol

- **Transport:** Unix stream socket, path **`/tmp/hashbang.sock`**.
- **Request:** one JSON object per connection (engine closes after one response):

```json
{
  "k": 10,
  "since": 300,
  "filter": ""
}
```

- `since` is the window size in **seconds** (e.g. `300` = 5 minutes). The CLI maps `5m` / `10m` / `30m` / `1h` to these values.
- **Response:** JSON array of results, sorted by descending estimated count:

```json
[
  { "tag": "golang", "count": 42 },
  { "tag": "ai", "count": 17 }
]
```

The CLI reads a **5s** deadline on the response.

---

## Configuration

Engine defaults are defined in [`engine/main.go`](engine/main.go) (`DefaultConfig()`):

| Setting | Default | Purpose |
|---------|---------|---------|
| `NumSlots` | `1000` | Ring buffer length for time slots. |
| `SlotSeconds` | `5` | Seconds per slot (must stay consistent with ingest/query semantics). |
| `MaxTop` | `100` | Max heap size per slot (candidate cap). |
| `CMSDepth` | `4` | CMS rows (hash functions). |
| `CMSWidth` | `2000` | CMS columns per row. |

Adjust these in code for different memory/latency tradeoffs.

---

## Testing

```bash
go test ./...
```

[`engine/e2e_test.go`](engine/e2e_test.go) covers ingest, rotation, aggregation, filtering, and heap behavior; [`engine/cms_test.go`](engine/cms_test.go) and [`engine/heap_test.go`](engine/heap_test.go) test building blocks.

---

## Implementation notes

- **CMS** ([`engine/cms.go`](engine/cms.go)) — FNV-based hashes; `Estimate` uses the minimum across rows (standard CMS estimate).
- **Min-heap** ([`engine/heap.go`](engine/heap.go)) — tracks the smallest count among the top candidates for eviction when the heap is full.
- **Generator** ([`generator/gentag.go`](generator/gentag.go)) — rotates through themed “trending” sets with noise; emits one word every **10ms** by default.
