// cmd/simulate/batch/main.go
package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lars-sto/error-recovery-simulation/internal/sim"
)

func main() {
	var (
		seed    = flag.Int64("seed", 1, "base seed (run seed = seed + i)")
		runs    = flag.Int("runs", 30, "repeats per scenario/mode")
		outPath = flag.String("out", "results/summary.csv", "output summary CSV file")
		filter  = flag.String("scenario", "", "scenario name filter (substring)")
		csvDir  = flag.String("csvdir", "", "optional: write per-run time series CSV into this directory (empty disables)")
		tsOnly  = flag.String("timeseries", "", "optional: comma-separated scenario substrings to write time series for (requires -csvdir)")
	)
	flag.Parse()

	scenarios := sim.DefaultScenarios(*seed)

	w, err := sim.NewSummaryCSVWriter(*outPath)
	if err != nil {
		panic(err)
	}
	defer func() { _ = w.Close() }()

	allowTS := parseCSVList(*tsOnly)

	for _, sc := range scenarios {
		if *filter != "" && !strings.Contains(sc.Name, *filter) {
			continue
		}

		for _, mode := range []sim.Mode{sim.ModeStatic, sim.ModeAdaptive} {
			for i := 0; i < *runs; i++ {
				runSeed := *seed + int64(i)

				// summary recorder (always)
				sumRec := sim.NewSummaryRecorder()

				// optional time series CSV recorder
				var rec sim.Recorder = sumRec
				if *csvDir != "" && wantTimeseries(sc.Name, allowTS) {
					path := filepath.Join(*csvDir, fmt.Sprintf("%s__%s__seed%d.csv", sc.Name, mode, runSeed))
					tsRec, err := sim.NewCSVRecorder(path)
					if err != nil {
						panic(err)
					}
					rec = sim.MultiRecorder(sumRec, tsRec)
				}

				res, err := sim.RunScenario(sc, sim.RunOptions{
					Mode:     mode,
					Seed:     runSeed,
					Recorder: rec,
				})
				if err != nil {
					panic(err)
				}

				row := sim.SummaryRow{
					Scenario:   sc.Name,
					Mode:       mode,
					Seed:       runSeed,
					DurationMs: res.Duration.Milliseconds(),

					FinalLossDeadline:   res.FinalLossDeadline,
					FinalLossNoDeadline: res.FinalLossNoDeadline,

					OverheadRatioBytes: res.OverheadRatioBytes,
					OverheadRatioPkts:  res.OverheadRatioPkts,

					MeanQueueDelayMs: sumRec.MeanQueueDelayMs(),

					MeanPolicyR:        sumRec.MeanPolicyR(),
					MaxPolicyR:         sumRec.MaxPolicyR(),
					MeanPolicyOverhead: sumRec.MeanPolicyOverhead(),

					MeanLossWindow: sumRec.MeanLossWindow(),
					MaxLossWindow:  sumRec.MaxLossWindow(),

					SentMediaPkts: res.SentMediaPkts,
					SentFECPkts:   res.SentFECPkts,
					DroppedMedia:  res.DroppedMediaPkts,
					DroppedFEC:    res.DroppedFECPkts,
					QueueDrops:    res.DroppedQueuePkts,
					WireDrops:     res.DroppedWirePkts,

					RecoveredPkts:      res.RecoveredPkts,
					UniquePkts:         res.UniquePkts,
					GoodWithinDeadline: res.GoodWithinDeadline,
				}

				if err := w.WriteRow(row); err != nil {
					panic(err)
				}
			}
		}
	}

	// ensure flushed
	if err := w.Close(); err != nil {
		panic(err)
	}
}

func parseCSVList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func wantTimeseries(name string, allow []string) bool {
	// empty allowlist => never write timeseries
	if len(allow) == 0 {
		return false
	}
	for _, sub := range allow {
		if strings.Contains(name, sub) {
			return true
		}
	}
	return false
}
