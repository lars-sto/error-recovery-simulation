package sim

import (
	"container/heap"
	"math"
	"time"

	"github.com/pion/rtp"
)

type DropReason string

const (
	DropNone     DropReason = ""
	DropQueue    DropReason = "queue_overflow"
	DropWireLoss DropReason = "wire_loss"
	DropZeroCap  DropReason = "zero_capacity"
)

type Link struct {
	spec  LinkSpec
	start time.Time

	nextAvail time.Time
	pq        eventHeap
}

type SendOutcome struct {
	Dropped    bool
	Reason     DropReason
	ArrivalAt  time.Time
	QueueDelay time.Duration
	SizeBytes  int
}

type DeliveredPacket struct {
	Pkt       rtp.Packet
	Arrives   time.Time
	SentAt    time.Time
	SizeBytes int
	IsFEC     bool
}

func NewLink(spec LinkSpec, start time.Time) *Link {
	l := &Link{spec: spec, start: start, nextAvail: start}
	heap.Init(&l.pq)
	return l
}

func (l *Link) Send(pkt rtp.Packet, sentAt time.Time, isFEC bool) SendOutcome {
	sizeBytes := pkt.MarshalSize()
	if sizeBytes <= 0 {
		sizeBytes = 12 + len(pkt.Payload)
	}

	capBps := math.Inf(1)
	if l.spec.CapacityBps != nil {
		capBps = l.spec.CapacityBps.At(sentAt.Sub(l.start))
	}
	if capBps == 0 {
		return SendOutcome{Dropped: true, Reason: DropZeroCap, SizeBytes: sizeBytes}
	}
	if capBps < 0 {
		capBps = 0
	}

	startTx := sentAt
	if l.nextAvail.After(startTx) {
		startTx = l.nextAvail
	}
	qDelay := startTx.Sub(sentAt)
	if l.spec.MaxQueueDelay > 0 && qDelay > l.spec.MaxQueueDelay {
		return SendOutcome{Dropped: true, Reason: DropQueue, QueueDelay: qDelay, SizeBytes: sizeBytes}
	}

	serSec := (float64(sizeBytes) * 8.0) / capBps
	if serSec < 0 {
		serSec = 0
	}
	ser := time.Duration(serSec * float64(time.Second))
	if ser == 0 && !math.IsInf(capBps, 1) {
		ser = time.Nanosecond
	}

	finishTx := startTx.Add(ser)
	l.nextAvail = finishTx

	arrival := finishTx.Add(l.spec.BaseOneWayDelay)
	if l.spec.Jitter > 0 {
		j := l.jitterFor(pkt.SSRC, pkt.SequenceNumber)
		arrival = arrival.Add(j)
	}

	if l.spec.Loss != nil {
		meta := PacketMeta{
			At:        sentAt.Sub(l.start),
			SSRC:      pkt.SSRC,
			PT:        pkt.PayloadType,
			Seq:       pkt.SequenceNumber,
			SizeBytes: sizeBytes,
			IsFEC:     isFEC,
		}
		if l.spec.Loss.Drop(meta) {
			return SendOutcome{Dropped: true, Reason: DropWireLoss, QueueDelay: qDelay, SizeBytes: sizeBytes}
		}
	}

	heap.Push(&l.pq, &deliveryEvent{
		at:        arrival,
		sentAt:    sentAt,
		pkt:       pkt,
		sizeBytes: sizeBytes,
		isFEC:     isFEC,
	})

	return SendOutcome{Dropped: false, Reason: DropNone, ArrivalAt: arrival, QueueDelay: qDelay, SizeBytes: sizeBytes}
}

func (l *Link) Next() (DeliveredPacket, bool) {
	if l.pq.Len() == 0 {
		return DeliveredPacket{}, false
	}
	ev := heap.Pop(&l.pq).(*deliveryEvent)
	return DeliveredPacket{Pkt: ev.pkt, Arrives: ev.at, SentAt: ev.sentAt, SizeBytes: ev.sizeBytes, IsFEC: ev.isFEC}, true
}

func (l *Link) jitterFor(ssrc uint32, seq uint16) time.Duration {
	u := u01(l.spec.Seed, ssrc, seq)
	x := (u * 2) - 1
	return time.Duration(x * float64(l.spec.Jitter))
}

type deliveryEvent struct {
	at        time.Time
	sentAt    time.Time
	pkt       rtp.Packet
	sizeBytes int
	isFEC     bool
}

type eventHeap []*deliveryEvent

func (h eventHeap) Len() int           { return len(h) }
func (h eventHeap) Less(i, j int) bool { return h[i].at.Before(h[j].at) }
func (h eventHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *eventHeap) Push(x any) { *h = append(*h, x.(*deliveryEvent)) }

func (h *eventHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
