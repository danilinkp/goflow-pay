package harness

import (
	"sort"
	"time"
)

type Overhead struct {
	TimerP50Ns       float64 `json:"timer_p50_ns"`
	TimerP99Ns       float64 `json:"timer_p99_ns"`
	HarnessNoopP50Ns float64 `json:"harness_noop_p50_ns"`
	HarnessNoopP99Ns float64 `json:"harness_noop_p99_ns"`
	LoopPerIterNs    float64 `json:"loop_per_iter_ns"`
}

var noopFn = emptyFn

//go:noinline
func emptyFn() {}

func Calibrate() Overhead {
	diffs := make([]int64, 1_000_000)
	for i := 0; i < 1_000_000; i++ {
		t0 := time.Now()
		t1 := time.Now()
		diffs[i] = t1.Sub(t0).Nanoseconds()
	}
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i] < diffs[j]
	})
	timerP50 := float64(diffs[len(diffs)/2])
	timerP99 := float64(diffs[int(float64(len(diffs))*0.99)])

	// Пустая операция занимает десятки наносекунд: за секунды набираются десятки миллионов
	// замеров по 24 байта, и контейнер упирается в лимит памяти. Миллиона замеров для p99 достаточно.
	cfg := Config{
		Warmup:   10 * time.Millisecond,
		Measure:  50 * time.Millisecond,
		Teardown: 10 * time.Millisecond,
		Ops:      []string{"noop"},
		Workers: []Worker{
			{
				Op: 0,
				Fn: func() {
					noopFn()
				},
			},
		},
	}
	res := Run(cfg)

	var measureLatencies []int64
	for _, s := range res.Samples {
		if s.Phase == PhaseMeasure {
			measureLatencies = append(measureLatencies, s.LatencyNs)
		}
	}

	var harnessP50, harnessP99 float64
	if len(measureLatencies) > 0 {
		sort.Slice(measureLatencies, func(i, j int) bool {
			return measureLatencies[i] < measureLatencies[j]
		})
		harnessP50 = float64(measureLatencies[len(measureLatencies)/2])
		harnessP99 = float64(measureLatencies[int(float64(len(measureLatencies))*0.99)])
	}

	for i := 0; i < 100_000; i++ {
		noopFn()
	}
	const iters = 10_000_000
	t0 := time.Now()
	for i := 0; i < iters; i++ {
		noopFn()
	}
	loopDur := time.Since(t0)
	loopPerIter := float64(loopDur.Nanoseconds()) / float64(iters)

	return Overhead{
		TimerP50Ns:       timerP50,
		TimerP99Ns:       timerP99,
		HarnessNoopP50Ns: harnessP50,
		HarnessNoopP99Ns: harnessP99,
		LoopPerIterNs:    loopPerIter,
	}
}
