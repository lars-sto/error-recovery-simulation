package sim

import (
	"math/rand"
	"sync"
	"time"
)

type PacketMeta struct {
	At        time.Duration
	SSRC      uint32
	PT        uint8
	Seq       uint16
	SizeBytes int
	IsFEC     bool
}

type LossModel interface {
	Name() string
	Drop(meta PacketMeta) bool
}

type ScheduledBernoulliLoss struct {
	Seed int64
	P    *FloatSchedule
	name string
}

func NewScheduledBernoulliLoss(name string, seed int64, p *FloatSchedule) *ScheduledBernoulliLoss {
	if name == "" {
		name = "bernoulli"
	}
	return &ScheduledBernoulliLoss{Seed: seed, P: p, name: name}
}

func (m *ScheduledBernoulliLoss) Name() string { return m.name }

func (m *ScheduledBernoulliLoss) Drop(meta PacketMeta) bool {
	p := 0.0
	if m.P != nil {
		p = m.P.At(meta.At)
	}
	if p <= 0 {
		return false
	}
	if p >= 1 {
		return true
	}
	return u01(m.Seed, meta.SSRC, meta.Seq) < p
}

type GilbertElliottLoss struct {
	NameStr string
	Seed    int64

	PGB float64
	PBG float64
	PG  float64
	PB  float64

	mu     sync.Mutex
	states map[uint32]*geState
}

type geState struct {
	bad bool
	r   *rand.Rand
}

func NewGilbertElliottLoss(name string, seed int64, pGB, pBG, pG, pB float64) *GilbertElliottLoss {
	if name == "" {
		name = "gilbert"
	}
	return &GilbertElliottLoss{
		NameStr: name, Seed: seed,
		PGB: pGB, PBG: pBG, PG: pG, PB: pB,
		states: make(map[uint32]*geState),
	}
}

func (m *GilbertElliottLoss) Name() string { return m.NameStr }

func (m *GilbertElliottLoss) Drop(meta PacketMeta) bool {
	st := m.state(meta.SSRC)

	if !st.bad {
		if st.r.Float64() < m.PGB {
			st.bad = true
		}
	} else {
		if st.r.Float64() < m.PBG {
			st.bad = false
		}
	}

	p := m.PG
	if st.bad {
		p = m.PB
	}
	if p <= 0 {
		return false
	}
	if p >= 1 {
		return true
	}
	return st.r.Float64() < p
}

func (m *GilbertElliottLoss) state(ssrc uint32) *geState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.states[ssrc]; ok {
		return st
	}
	const mix uint64 = 0x9e3779b97f4a7c15

	u := uint64(m.Seed) ^ (uint64(ssrc) * mix)
	seed := int64(u)
	st := &geState{r: rand.New(rand.NewSource(seed))}
	m.states[ssrc] = st
	return st
}
