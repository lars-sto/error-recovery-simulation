package sim

import "time"

func RandomLossScenario(name string, p float64, dur time.Duration, seed int64) Scenario {
	return Scenario{
		Name:         name,
		Duration:     dur,
		Seed:         seed,
		PacketRateHz: 100,
		PayloadSize:  900,

		SSRCMedia: 1111,
		SSRCFEC:   2222,
		PTMedia:   96,
		PTFEC:     97,

		Network: NetworkSpec{
			RandomLoss: p,
		},
	}
}
