package sim

import "time"

type RunResult struct {
	Scenario string
	Mode     Mode
	Seed     int64

	Duration time.Duration

	SentMediaPkts  int64
	SentFECPkts    int64
	SentMediaBytes int64
	SentFECBytes   int64

	DroppedMediaPkts int64
	DroppedFECPkts   int64
	DroppedQueuePkts int64
	DroppedWirePkts  int64

	RecvMediaPkts int64
	RecvFECPkts   int64

	RecoveredPkts int64
	UniquePkts    int64

	GoodWithinDeadline  int64
	FinalLossNoDeadline float64
	FinalLossDeadline   float64

	OverheadRatioPkts  float64
	OverheadRatioBytes float64
}
