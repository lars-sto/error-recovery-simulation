// sim/impair.go (fortsetzung)
package sim

import (
	"math/rand"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

type ImpairingRTPWriter struct {
	Next interceptor.RTPWriter
	Loss LossModel

	// Optional: Counters/Stats
	DropCount uint64
}

func (w *ImpairingRTPWriter) Write(h *rtp.Header, payload []byte, a interceptor.Attributes) (int, error) {
	if w.Loss != nil && w.Loss.Drop(time.Now(), h.SSRC, h.SequenceNumber, h.PayloadType, false) {
		// drop silently: pretend it was sent
		w.DropCount++
		return len(payload), nil
	}
	return w.Next.Write(h, payload, a)
}

// Gilbert-Elliot
type GilbertElliottLoss struct {
	pGB float64
	pBG float64
	pG  float64
	pB  float64
	bad bool
	r   *rand.Rand
}

func NewGilbertElliottLoss(pGB, pBG, pG, pB float64, seed int64) *GilbertElliottLoss {
	return &GilbertElliottLoss{
		pGB: pGB, pBG: pBG, pG: pG, pB: pB,
		r: rand.New(rand.NewSource(seed)),
	}
}

func (m *GilbertElliottLoss) Name() string { return "gilbert-elliott" }

func (m *GilbertElliottLoss) Drop(_ time.Time, _ uint32, _ uint16, _ uint8, _ bool) bool {
	// transition
	if !m.bad {
		if m.r.Float64() < m.pGB {
			m.bad = true
		}
	} else {
		if m.r.Float64() < m.pBG {
			m.bad = false
		}
	}
	// emit drop
	p := m.pG
	if m.bad {
		p = m.pB
	}
	if p <= 0 {
		return false
	}
	if p >= 1 {
		return true
	}
	return m.r.Float64() < p
}

// Random Loss
type RandomLoss struct {
	P float64 // 0..1
	r *rand.Rand
}

func NewRandomLoss(p float64, seed int64) *RandomLoss {
	return &RandomLoss{P: p, r: rand.New(rand.NewSource(seed))}
}

func (m *RandomLoss) Name() string { return "random" }

func (m *RandomLoss) Drop(_ time.Time, _ uint32, _ uint16, _ uint8, _ bool) bool {
	if m.P <= 0 {
		return false
	}
	if m.P >= 1 {
		return true
	}
	return m.r.Float64() < m.P
}
