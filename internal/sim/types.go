package sim

import "time"

type Mode string

const (
	ModeStatic   Mode = "static_flexfec"
	ModeAdaptive Mode = "adaptive_engine"
)

type RTPIDs struct {
	MediaSSRC uint32
	FECSSRC   uint32
	MediaPT   uint8
	FECPT     uint8
}

type SenderSpec struct {
	PacketRateHz  int
	PayloadBytes  int
	StartSeq      uint16
	StartTS       uint32
	TimestampStep uint32
	StartTime     time.Time
}

func (s SenderSpec) Interval() time.Duration {
	if s.PacketRateHz <= 0 {
		return 0
	}
	return time.Second / time.Duration(s.PacketRateHz)
}

func (s SenderSpec) MediaBitrateBps(includeRTPHeader bool) float64 {
	if s.PacketRateHz <= 0 || s.PayloadBytes <= 0 {
		return 0
	}
	bytesPerPkt := float64(s.PayloadBytes)
	if includeRTPHeader {
		bytesPerPkt += 12
	}
	return bytesPerPkt * 8 * float64(s.PacketRateHz)
}

type LinkSpec struct {
	BaseOneWayDelay time.Duration
	Jitter          time.Duration
	MaxQueueDelay   time.Duration
	CapacityBps     *FloatSchedule
	Loss            LossModel
	Seed            int64
}

type Scenario struct {
	Name     string
	Duration time.Duration

	IDs    RTPIDs
	Sender SenderSpec

	K       uint32
	StaticR uint32

	StatsInterval   time.Duration
	BWE             *FloatSchedule
	RTTMs           int
	JitterMs        int
	PlayoutDeadline time.Duration

	Link LinkSpec

	Seed int64
}

type FloatSchedule struct {
	Points  []FloatPoint
	Default float64
}

type FloatPoint struct {
	At    time.Duration
	Value float64
}

func (s *FloatSchedule) At(t time.Duration) float64 {
	if s == nil || len(s.Points) == 0 {
		if s == nil {
			return 0
		}
		return s.Default
	}
	v := s.Points[0].Value
	for i := 0; i < len(s.Points); i++ {
		if t < s.Points[i].At {
			break
		}
		v = s.Points[i].Value
	}
	return v
}
