package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
)

type nopSink struct{}

func (nopSink) Publish(recovery.PolicyDecision) {}

// Choose scenario here (single source of truth).
const (
	scenarioName = "03_bwe_bottleneck"
	// scenarioName = "01_loss_increase"
	// scenarioName = "02_loss_threshold"
)

func main() {
	start := time.Now()

	series := pickScenario(scenarioName, start)
	outPath := fmt.Sprintf("simdata/%s.csv", scenarioName)

	cfg := recovery.DefaultConfig()

	obs, err := NewCSVObserver(outPath)
	if err != nil {
		panic(err)
	}
	defer func() { _ = obs.Close() }()

	statsCh := make(chan recovery.NetworkStats, len(series))
	for _, s := range series {
		statsCh <- s
	}
	close(statsCh)

	src := recovery.NewChanSource(statsCh)

	eng := recovery.NewEngine(cfg, src, nopSink{}, obs)
	eng.Run(context.Background())
}

func pickScenario(name string, start time.Time) []recovery.NetworkStats {
	switch name {
	case "01_loss_increase":
		return scenario01IncreasingLoss(start)
	case "02_loss_threshold_oscillation":
		return scenario02LossAroundEnable(start)
	case "03_bwe_bottleneck":
		return scenario03BWEBottleneck(start)
	default:
		panic("unknown scenario: " + name)
	}
}
