package sim

import (
	"time"

	"github.com/pion/rtp"
)

type Receiver struct {
	decoder *FlexFEC03Decoder
	availAt map[uint16]time.Time

	recvMedia int64
	recvFEC   int64
	recovered int64

	mediaSSRC uint32
	fecSSRC   uint32
	mediaPT   uint8
	fecPT     uint8
}

func NewReceiver(ids RTPIDs) *Receiver {
	return &Receiver{
		decoder:   NewFlexFEC03Decoder(ids.FECSSRC, ids.MediaSSRC),
		availAt:   make(map[uint16]time.Time, 4096),
		mediaSSRC: ids.MediaSSRC,
		fecSSRC:   ids.FECSSRC,
		mediaPT:   ids.MediaPT,
		fecPT:     ids.FECPT,
	}
}

func (r *Receiver) OnPacket(pkt rtp.Packet, at time.Time) {
	isFEC := (pkt.SSRC == r.fecSSRC) || (pkt.PayloadType == r.fecPT)
	if isFEC {
		r.recvFEC++
	} else {
		r.recvMedia++
		r.markAvailable(pkt.SequenceNumber, at)
	}

	recovered := r.decoder.Push(pkt)
	for _, rp := range recovered {
		if rp.SSRC != r.mediaSSRC {
			continue
		}
		if r.markAvailable(rp.SequenceNumber, at) {
			r.recovered++
		}
	}
}

func (r *Receiver) markAvailable(seq uint16, at time.Time) bool {
	if _, ok := r.availAt[seq]; ok {
		return false
	}
	r.availAt[seq] = at
	return true
}

type ReceiverSnapshot struct {
	RecvMedia int64
	RecvFEC   int64
	Recovered int64
	Unique    int64
}

func (r *Receiver) Snapshot() ReceiverSnapshot {
	return ReceiverSnapshot{
		RecvMedia: r.recvMedia,
		RecvFEC:   r.recvFEC,
		Recovered: r.recovered,
		Unique:    int64(len(r.availAt)),
	}
}
