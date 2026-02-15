package adapter

import (
	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
	"github.com/pion/interceptor/pkg/flexfec"
)

type FlexFECAdapter struct {
	Bus *RuntimeBus
}

func NewFlexFECAdapter(bus *RuntimeBus) *FlexFECAdapter {
	return &FlexFECAdapter{Bus: bus}
}

func (a *FlexFECAdapter) Apply(mediaSSRC uint32, d recovery.PolicyDecision) {
	f := d.FEC

	cfg := flexfec.RuntimeConfig{
		Enabled:          f.Enabled,
		NumMediaPackets:  f.NumMediaPackets,
		NumFECPackets:    f.NumFECPackets,
		CoverageMode:     flexfec.CoverageMode(string(f.CoverageMode)),
		InterleaveStride: f.InterleaveStride,
		BurstSpan:        f.BurstSpan,
	}

	a.Bus.Publish(mediaSSRC, cfg)
}
