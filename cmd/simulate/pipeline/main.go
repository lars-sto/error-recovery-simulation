package main

import (
	"fmt"

	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
	"github.com/lars-sto/error-recovery-simulation/internal/adapter"
	"github.com/pion/interceptor/pkg/flexfec"
)

func main() {
	bus := adapter.NewRuntimeBus()
	ffc := adapter.NewFlexFECAdapter(bus)

	// build pion interceptor chain
	fecFactory, _ := flexfec.NewFecInterceptor(
		flexfec.WithConfigSource(bus),
		// defaults are fallback only:
		flexfec.NumMediaPackets(10),
		flexfec.NumFECPackets(2),
	)
	fmt.Print(fecFactory)

	// when you know the mediaSSRC for the outbound stream:
	sink := adapter.SinkFunc(func(d recovery.PolicyDecision) {
		ffc.Apply(mediaSSRC, d)
	})

	eng := recovery.NewEngine(recovery.DefaultConfig(), statsSource, sink, obs)
	go eng.Run(ctx)
}
