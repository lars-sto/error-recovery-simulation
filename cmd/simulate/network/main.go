package main

import (
	"flag"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/lars-sto/error-recovery-simulation/internal/sim"
)

func main() {
	var (
		seed     = flag.Int64("seed", 1, "base seed")
		runs     = flag.Int("runs", 1, "repeats per scenario/mode")
		outDir   = flag.String("out", "", "output directory for per-run CSV time series")
		writeCSV = flag.Bool("csv", false, "write per-run time series CSV")
		filter   = flag.String("scenario", "", "scenario name filter (substring)")
	)
	flag.Parse()

	scenarios := sim.DefaultScenarios(*seed)

	for _, sc := range scenarios {
		if *filter != "" && !strings.Contains(sc.Name, *filter) {
			continue
		}

		for _, mode := range []sim.Mode{sim.ModeStatic, sim.ModeAdaptive} {
			var losses []float64
			var overhead []float64

			for i := 0; i < *runs; i++ {
				runSeed := *seed + int64(i)

				var rec sim.Recorder
				if *writeCSV && *outDir != "" {
					path := filepath.Join(*outDir, fmt.Sprintf("%s__%s__seed%d.csv", sc.Name, mode, runSeed))
					r, err := sim.NewCSVRecorder(path)
					if err != nil {
						panic(err)
					}
					rec = r
				}

				res, err := sim.RunScenario(sc, sim.RunOptions{Mode: mode, Seed: runSeed, Recorder: rec})
				if err != nil {
					panic(err)
				}

				fmt.Printf("%s | %s | seed=%d | sentMedia=%d sentFEC=%d recvMedia=%d recvFEC=%d recovered=%d unique=%d good=%d loss=%.3f loss_deadline=%.3f oh_bytes=%.3f\n",
					sc.Name, mode, runSeed,
					res.SentMediaPkts, res.SentFECPkts,
					res.RecvMediaPkts, res.RecvFECPkts,
					res.RecoveredPkts, res.UniquePkts, res.GoodWithinDeadline,
					res.FinalLossNoDeadline, res.FinalLossDeadline,
					res.OverheadRatioBytes,
				)

				losses = append(losses, res.FinalLossDeadline)
				overhead = append(overhead, res.OverheadRatioBytes)
			}

			if len(losses) > 1 {
				fmt.Printf("%s | %s | mean_loss_deadline=%.3f std=%.3f | mean_oh_bytes=%.3f\n",
					sc.Name, mode, mean(losses), stddev(losses), mean(overhead),
				)
			}
		}
	}
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func stddev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	v := 0.0
	for _, x := range xs {
		d := x - m
		v += d * d
	}
	v /= float64(len(xs) - 1)
	return math.Sqrt(v)
}
