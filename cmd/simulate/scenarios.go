package main

import (
	"strconv"
	"time"

	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
)

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}

// Simple Szenario for monotonically increasing FEC protection
func scenario01IncreasingLoss(start time.Time) []recovery.NetworkStats {
	var stats []recovery.NetworkStats

	for i := 0; i < 100; i++ {
		stats = append(stats, recovery.NetworkStats{
			Timestamp:      start.Add(time.Duration(i) * time.Second),
			LossRate:       float64(i) / 100.0 * 0.15, // 0 → 15%
			RTTMs:          100,
			CurrentBitrate: 1_000_000,
			TargetBitrate:  1_200_000,
		})
	}

	return stats
}

// Hysterese
func scenario02LossAroundEnable(start time.Time) []recovery.NetworkStats {
	var stats []recovery.NetworkStats

	lossPattern := []float64{
		0.040, 0.045, 0.050, 0.055, 0.060,
		0.055, 0.050, 0.045, 0.040,
		0.045, 0.050, 0.055,
		0.050, 0.045, 0.040,
	}

	for i, loss := range lossPattern {
		stats = append(stats, recovery.NetworkStats{
			Timestamp:      start.Add(time.Duration(i) * time.Second),
			LossRate:       loss,
			RTTMs:          100,
			CurrentBitrate: 1_000_000,
			TargetBitrate:  1_200_000,
		})
	}

	return stats
}

// BWE bottleneck
func scenario03BWEBottleneck(start time.Time) []recovery.NetworkStats {
	var stats []recovery.NetworkStats

	type step struct {
		loss   float64
		target float64
	}

	steps := []step{
		{loss: 0.08, target: 1_300_000}, // plenty of headroom
		{loss: 0.08, target: 1_200_000},
		{loss: 0.08, target: 1_100_000},
		{loss: 0.08, target: 1_050_000}, // tight: ~5% overhead
		{loss: 0.08, target: 1_020_000}, // very tight: ~2% overhead
		{loss: 0.08, target: 1_000_000}, // zero headroom: overhead -> 0
		{loss: 0.08, target: 980_000},   // below current: still 0
		{loss: 0.08, target: 1_050_000}, // recover a bit
		{loss: 0.08, target: 1_200_000}, // recover more
	}

	cur := 1_000_000.0

	for i := 0; i < len(steps); i++ {
		s := steps[i]
		for k := 0; k < 10; k++ {
			stats = append(stats, recovery.NetworkStats{
				Timestamp:      start.Add(time.Duration(i*10+k) * time.Second),
				LossRate:       s.loss,
				RTTMs:          200,
				CurrentBitrate: cur,
				TargetBitrate:  s.target,
			})
		}
	}

	return stats
}
