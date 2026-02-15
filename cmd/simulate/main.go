package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
	"github.com/lars-sto/adaptive-fec-simulation/internal/adapter"
	"github.com/pion/interceptor/pkg/flexfec"
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
	case "02_loss_threshold":
		return scenario02LossAroundEnable(start)
	case "03_bwe_bottleneck":
		return scenario03BWEBottleneck(start)
	default:
		panic("unknown scenario: " + name)
	}
}

func wiring() {
	bus := adapter.NewRuntimeBus()
	ffc := adapter.NewFlexFECAdapter(bus)

	// build pion interceptor chain
	fecFactory, _ := flexfec.NewFecInterceptor(
		flexfec.WithConfigSource(bus),
		// defaults are fallback only:
		flexfec.NumMediaPackets(10),
		flexfec.NumFECPackets(2),
	)

	// when you know the mediaSSRC for the outbound stream:
	sink := adapter.SinkFunc(func(d recovery.PolicyDecision) {
		ffc.Apply(mediaSSRC, d)
	})

	eng := recovery.NewEngine(recovery.DefaultConfig(), statsSource, sink, obs)
	go eng.Run(ctx)
}
