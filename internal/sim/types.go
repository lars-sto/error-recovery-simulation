package sim

import "time"

type Mode string

const (
	ModeStatic   Mode = "static_flexfec"
	ModeAdaptive Mode = "adaptive_engine"
)

type NetworkSpec struct {
	RandomLoss float64 // 0..1
	// später: burst params, reorder, jitter, rate cap, etc.
}

type RunnerConfig struct {
	OutDir string
}

type Scenario struct {
	Name string

	Duration time.Duration

	// Network behavior
	Loss         LossModel
	OneWayDelay  time.Duration // optional, later (if vnet supports link delay)
	RateLimitBps int           // optional, later

	// Media/FEC identifiers (so logger can classify)
	SSRCMedia uint32
	SSRCFEC   uint32
	PTMedia   uint8
	PTFEC     uint8

	Mode Mode // StaticFlexFEC or AdaptiveEngine etc.
}

type LossModel interface {
	Drop(now time.Time, meta PacketMeta) bool
	Name() string
}

type PacketMeta struct {
	SSRC uint32
	PT   uint8
	Seq  uint16
	TS   uint32
	Len  int
}
