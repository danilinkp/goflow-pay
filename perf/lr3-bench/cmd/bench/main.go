package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"lr3bench/internal/harness"
	"lr3bench/internal/scenario"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

type MetaJSON struct {
	GoVersion       string                    `json:"go_version"`
	Scenario        string                    `json:"scenario"`
	Size            int                       `json:"size"`
	BatchSize       int                       `json:"batch_size"`
	Run             int                       `json:"run"`
	Seed            uint64                    `json:"seed"`
	WarmupNs        int64                     `json:"warmup_ns"`
	MeasureNs       int64                     `json:"measure_ns"`
	TeardownNs      int64                     `json:"teardown_ns"`
	TotalDurationNs int64                     `json:"total_duration_ns"`
	Counts          map[string]map[string]int `json:"counts"`
	GCCyclesMeasure uint32                    `json:"gc_cycles_measure"`
	Overhead        harness.Overhead          `json:"overhead"`
	Env             EnvJSON                   `json:"env"`
}

type EnvJSON struct {
	Gomaxprocs int    `json:"gomaxprocs"`
	NumCPU     int    `json:"num_cpu"`
	StartedAt  string `json:"started_at"`
}

func main() {
	scenarioFlag := flag.String("scenario", "", "scenario name")
	sizeFlag := flag.Int("size", 100000, "map size")
	outFlag := flag.String("out", "", "output directory")
	seedFlag := flag.Uint64("seed", 42, "random seed")
	warmupFlag := flag.Duration("warmup", 2*time.Second, "warmup duration")
	durationFlag := flag.Duration("duration", 10*time.Second, "measurement duration")
	teardownFlag := flag.Duration("teardown", 1*time.Second, "teardown duration")
	runFlag := flag.Int("run", 1, "run index")
	liveFlag := flag.Duration("live", 0, "interval of live histogram output to stdout, 0 disables it")
	flag.Parse()

	if *scenarioFlag == "" || *outFlag == "" {
		panic("flags -scenario and -out are required")
	}

	startedAt := time.Now().UTC().Format(time.RFC3339)
	overhead := harness.Calibrate()

	cfg, validateFn, err := scenario.Build(*scenarioFlag, *sizeFlag, *seedFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build scenario: %v\n", err)
		os.Exit(1)
	}
	cfg.Warmup = *warmupFlag
	cfg.Measure = *durationFlag
	cfg.Teardown = *teardownFlag
	if *liveFlag > 0 {
		cfg.Live = harness.NewLive(os.Stdout, *liveFlag, cfg.Ops)
	}

	res := harness.Run(cfg)

	if validateFn != nil {
		if err := validateFn(); err != nil {
			fmt.Fprintf(os.Stderr, "validate scenario: %v\n", err)
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(*outFlag, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir out: %v\n", err)
		os.Exit(1)
	}

	csvFile, err := os.Create(filepath.Join(*outFlag, "samples.csv"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create samples.csv: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	bw := bufio.NewWriter(csvFile)
	bw.WriteString("start_ns,latency_ns,op,phase\n")

	counts := map[string]map[string]int{
		"warmup":   make(map[string]int),
		"measure":  make(map[string]int),
		"teardown": make(map[string]int),
	}
	for _, opName := range cfg.Ops {
		counts["warmup"][opName] = 0
		counts["measure"][opName] = 0
		counts["teardown"][opName] = 0
	}

	var numBuf []byte
	for _, s := range res.Samples {
		var phaseStr string
		switch s.Phase {
		case harness.PhaseWarmup:
			phaseStr = "warmup"
		case harness.PhaseMeasure:
			phaseStr = "measure"
		case harness.PhaseTeardown:
			phaseStr = "teardown"
		}

		opName := cfg.Ops[s.Op]
		counts[phaseStr][opName]++

		numBuf = strconv.AppendInt(numBuf[:0], s.StartNs, 10)
		bw.Write(numBuf)
		bw.WriteByte(',')
		numBuf = strconv.AppendInt(numBuf[:0], s.LatencyNs, 10)
		bw.Write(numBuf)
		bw.WriteByte(',')
		bw.WriteString(opName)
		bw.WriteByte(',')
		bw.WriteString(phaseStr)
		bw.WriteByte('\n')
	}
	if err := bw.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush samples.csv: %v\n", err)
		os.Exit(1)
	}

	metaSize := *sizeFlag
	if *scenarioFlag == "grow" {
		metaSize = 1000
	}

	meta := MetaJSON{
		GoVersion:       runtime.Version(),
		Scenario:        *scenarioFlag,
		Size:            metaSize,
		BatchSize:       scenario.BatchSize,
		Run:             *runFlag,
		Seed:            *seedFlag,
		WarmupNs:        cfg.Warmup.Nanoseconds(),
		MeasureNs:       cfg.Measure.Nanoseconds(),
		TeardownNs:      cfg.Teardown.Nanoseconds(),
		TotalDurationNs: res.TotalDuration.Nanoseconds(),
		Counts:          counts,
		GCCyclesMeasure: res.GCCycles,
		Overhead:        overhead,
		Env: EnvJSON{
			Gomaxprocs: runtime.GOMAXPROCS(0),
			NumCPU:     runtime.NumCPU(),
			StartedAt:  startedAt,
		},
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal meta.json: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*outFlag, "meta.json"), metaData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write meta.json: %v\n", err)
		os.Exit(1)
	}
}
