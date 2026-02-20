package sim

import "math"

// splitmix64: guter deterministischer 64-bit mixer
func splitmix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// u01 returns a deterministic float in [0,1) from (seed,ssrc,seq)
func u01(seed int64, ssrc uint32, seq uint16) float64 {
	// pack inputs
	x := uint64(seed)
	x ^= uint64(ssrc)<<32 | uint64(seq)

	y := splitmix64(x)

	// take top 53 bits -> float64 in [0,1)
	v := float64(y>>11) / (1 << 53)
	if v >= 1 {
		return math.Nextafter(1, 0)
	}
	return v
}
