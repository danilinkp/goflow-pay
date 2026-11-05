package scenario

import (
	"fmt"
	"lr3bench/internal/harness"
	"math/rand/v2"
	"sync"
)

const BatchSize = 1000

var Sink uint64

var mixedSinks [3]uint64

func Build(name string, size int, seed uint64) (harness.Config, func() error, error) {
	rng := rand.New(rand.NewPCG(seed, seed))

	if name == "grow" {
		growKeys := make([]uint64, BatchSize)
		seen := make(map[uint64]struct{}, BatchSize)
		for len(seen) < BatchSize {
			k := rng.Uint64()
			if _, exists := seen[k]; exists {
				continue
			}
			growKeys[len(seen)] = k
			seen[k] = struct{}{}
		}

		worker := harness.Worker{
			Op: 0,
			Fn: func() {
				m := make(map[uint64]uint64)
				for i := 0; i < BatchSize; i++ {
					m[growKeys[i]] = growKeys[i]
				}
				Sink = uint64(len(m))
			},
		}
		return harness.Config{
			Ops:     []string{"grow"},
			Workers: []harness.Worker{worker},
		}, nil, nil
	}

	present := make([]uint64, 0, size)
	seen := make(map[uint64]struct{}, size*3)
	for len(present) < size {
		k := rng.Uint64()
		if _, exists := seen[k]; exists {
			continue
		}
		seen[k] = struct{}{}
		present = append(present, k)
	}

	m := make(map[uint64]uint64)
	for _, k := range present {
		m[k] = k
	}

	absent := make([]uint64, 0, size)
	for len(absent) < size {
		k := rng.Uint64()
		if _, exists := seen[k]; exists {
			continue
		}
		seen[k] = struct{}{}
		absent = append(absent, k)
	}

	spare := make([]uint64, 0, size)
	for len(spare) < size {
		k := rng.Uint64()
		if _, exists := seen[k]; exists {
			continue
		}
		seen[k] = struct{}{}
		spare = append(spare, k)
	}

	shuffledPresent := make([]uint64, len(present))
	copy(shuffledPresent, present)
	rng.Shuffle(len(shuffledPresent), func(i, j int) {
		shuffledPresent[i], shuffledPresent[j] = shuffledPresent[j], shuffledPresent[i]
	})

	shuffledAbsent := make([]uint64, len(absent))
	copy(shuffledAbsent, absent)
	rng.Shuffle(len(shuffledAbsent), func(i, j int) {
		shuffledAbsent[i], shuffledAbsent[j] = shuffledAbsent[j], shuffledAbsent[i]
	})

	switch name {
	case "lookup_hit":
		cursor := 0
		nKeys := len(shuffledPresent)
		worker := harness.Worker{
			Op: 0,
			Fn: func() {
				idx := cursor
				var s uint64
				for i := 0; i < BatchSize; i++ {
					v, ok := m[shuffledPresent[idx]]
					if ok {
						s += v
					}
					idx++
					if idx == nKeys {
						idx = 0
					}
				}
				Sink = s
				cursor = idx
			},
		}
		return harness.Config{
			Ops:     []string{"lookup_hit"},
			Workers: []harness.Worker{worker},
		}, nil, nil

	case "lookup_miss":
		cursor := 0
		nKeys := len(shuffledAbsent)
		worker := harness.Worker{
			Op: 0,
			Fn: func() {
				idx := cursor
				var s uint64
				for i := 0; i < BatchSize; i++ {
					v, ok := m[shuffledAbsent[idx]]
					if ok {
						s += v
					}
					idx++
					if idx == nKeys {
						idx = 0
					}
				}
				Sink = s
				cursor = idx
			},
		}
		return harness.Config{
			Ops:     []string{"lookup_miss"},
			Workers: []harness.Worker{worker},
		}, nil, nil

	case "update":
		cursor := 0
		nKeys := len(shuffledPresent)
		worker := harness.Worker{
			Op: 0,
			Fn: func() {
				idx := cursor
				for i := 0; i < BatchSize; i++ {
					k := shuffledPresent[idx]
					m[k] = k + 1
					idx++
					if idx == nKeys {
						idx = 0
					}
				}
				cursor = idx
			},
		}
		return harness.Config{
			Ops:     []string{"update"},
			Workers: []harness.Worker{worker},
		}, nil, nil

	case "churn":
		activeKeys := make([]uint64, size)
		copy(activeKeys, present)
		spareKeys := make([]uint64, size)
		copy(spareKeys, spare)

		delIdx := 0
		insIdx := 0
		worker := harness.Worker{
			Op: 0,
			Fn: func() {
				for i := 0; i < 500; i++ {
					kDel := activeKeys[delIdx]
					delete(m, kDel)
					kIns := spareKeys[insIdx]
					m[kIns] = kIns

					activeKeys[delIdx] = kIns
					spareKeys[insIdx] = kDel

					delIdx++
					if delIdx == size {
						delIdx = 0
					}
					insIdx++
					if insIdx == size {
						insIdx = 0
					}
				}
			},
		}
		validateFn := func() error {
			if len(m) != size {
				return fmt.Errorf("churn map size corrupted: got %d, want %d", len(m), size)
			}
			return nil
		}
		return harness.Config{
			Ops:     []string{"churn"},
			Workers: []harness.Worker{worker},
		}, validateFn, nil

	case "mixed":
		var mu sync.RWMutex

		activeKeys := make([]uint64, size)
		copy(activeKeys, present)
		spareKeys := make([]uint64, size)
		copy(spareKeys, spare)

		delIdx := 0
		insIdx := 0
		writeFn := func() {
			mu.Lock()
			for i := 0; i < 100; i++ {
				kDel := activeKeys[delIdx]
				delete(m, kDel)
				kIns := spareKeys[insIdx]
				m[kIns] = kIns

				activeKeys[delIdx] = kIns
				spareKeys[insIdx] = kDel

				delIdx++
				if delIdx == size {
					delIdx = 0
				}
				insIdx++
				if insIdx == size {
					insIdx = 0
				}
			}
			mu.Unlock()
		}

		workers := make([]harness.Worker, 4)
		for r := 0; r < 3; r++ {
			readerIdx := r
			rKeys := make([]uint64, len(present))
			copy(rKeys, present)
			rng.Shuffle(len(rKeys), func(i, j int) {
				rKeys[i], rKeys[j] = rKeys[j], rKeys[i]
			})
			rCursor := 0
			nRKeys := len(rKeys)

			workers[r] = harness.Worker{
				Op: 0,
				Fn: func() {
					mu.RLock()
					idx := rCursor
					var s uint64
					for i := 0; i < BatchSize; i++ {
						v, ok := m[rKeys[idx]]
						if ok {
							s += v
						}
						idx++
						if idx == nRKeys {
							idx = 0
						}
					}
					mixedSinks[readerIdx] = s
					rCursor = idx
					mu.RUnlock()
				},
			}
		}
		workers[3] = harness.Worker{
			Op: 1,
			Fn: writeFn,
		}

		validateFn := func() error {
			if len(m) != size {
				return fmt.Errorf("mixed map size corrupted: got %d, want %d", len(m), size)
			}
			return nil
		}

		return harness.Config{
			Ops:     []string{"mixed_read", "mixed_write"},
			Workers: workers,
		}, validateFn, nil

	default:
		return harness.Config{}, nil, fmt.Errorf("unknown scenario: %s", name)
	}
}
