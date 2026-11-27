package harness

import (
	"runtime"
	"sort"
	"sync"
	"time"
)

type Phase uint8

const (
	PhaseWarmup Phase = iota
	PhaseMeasure
	PhaseTeardown
)

type Sample struct {
	StartNs   int64
	LatencyNs int64
	Op        uint8
	Phase     Phase
}

type Worker struct {
	Op uint8
	Fn func()
}

type Config struct {
	Warmup   time.Duration
	Measure  time.Duration
	Teardown time.Duration
	Ops      []string
	Workers  []Worker
	Live     *Live
}

type Result struct {
	Samples       []Sample
	TotalDuration time.Duration
	GCCycles      uint32
}

func Run(cfg Config) Result {
	runtime.GC()

	barrier := make(chan struct{})
	workerSamples := make([][]Sample, len(cfg.Workers))

	var startWg sync.WaitGroup
	startWg.Add(len(cfg.Workers))

	var doneWg sync.WaitGroup
	doneWg.Add(len(cfg.Workers))

	warmupEnd := cfg.Warmup
	measureEnd := cfg.Warmup + cfg.Measure
	totalEnd := cfg.Warmup + cfg.Measure + cfg.Teardown

	var start time.Time

	for i, w := range cfg.Workers {
		idx := i
		worker := w
		workerSamples[idx] = make([]Sample, 0, 1<<21)

		go func() {
			defer doneWg.Done()
			samples := workerSamples[idx]
			startWg.Done()
			<-barrier

			for {
				t0 := time.Now()
				worker.Fn()
				t1 := time.Now()

				if t1.Sub(start) > totalEnd {
					break
				}

				offset := t0.Sub(start)
				var phase Phase
				if offset < warmupEnd {
					phase = PhaseWarmup
				} else if offset < measureEnd {
					phase = PhaseMeasure
				} else {
					phase = PhaseTeardown
				}

				latency := t1.Sub(t0).Nanoseconds()
				samples = append(samples, Sample{
					StartNs:   offset.Nanoseconds(),
					LatencyNs: latency,
					Op:        worker.Op,
					Phase:     phase,
				})
				if cfg.Live != nil {
					cfg.Live.record(worker.Op, phase, latency)
				}
			}
			workerSamples[idx] = samples
		}()
	}

	startWg.Wait()

	var gcBefore, gcAfter uint32
	var gcWg sync.WaitGroup
	gcWg.Add(1)

	start = time.Now()
	close(barrier)

	liveStop := make(chan struct{})
	liveDone := make(chan struct{})
	if cfg.Live != nil {
		go cfg.Live.stream(liveStop, liveDone)
	}

	go func() {
		defer gcWg.Done()

		sleepWarmup := start.Add(warmupEnd).Sub(time.Now())
		if sleepWarmup > 0 {
			time.Sleep(sleepWarmup)
		}
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		gcBefore = m1.NumGC

		sleepMeasure := start.Add(measureEnd).Sub(time.Now())
		if sleepMeasure > 0 {
			time.Sleep(sleepMeasure)
		}
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		gcAfter = m2.NumGC
	}()

	doneWg.Wait()
	gcWg.Wait()

	totalDuration := time.Since(start)

	if cfg.Live != nil {
		close(liveStop)
		<-liveDone
	}

	totalCount := 0
	for _, s := range workerSamples {
		totalCount += len(s)
	}

	merged := make([]Sample, 0, totalCount)
	for _, s := range workerSamples {
		merged = append(merged, s...)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].StartNs < merged[j].StartNs
	})

	var gcCycles uint32
	if gcAfter >= gcBefore {
		gcCycles = gcAfter - gcBefore
	}

	return Result{
		Samples:       merged,
		TotalDuration: totalDuration,
		GCCycles:      gcCycles,
	}
}
