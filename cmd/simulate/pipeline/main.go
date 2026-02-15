package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lars-sto/error-recovery-simulation/internal/adapter"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/flexfec"
	"github.com/pion/rtp"
)

func main() {
	const (
		mediaSSRC uint32 = 1111
		fecSSRC   uint32 = 2222
		mediaPT   uint8  = 96
		fecPT     uint8  = 97
	)

	bus := adapter.NewRuntimeBus()

	reg := &interceptor.Registry{}
	fecFactory, err := flexfec.NewFecInterceptor(
		flexfec.WithConfigSource(bus),
		flexfec.NumMediaPackets(10),
		flexfec.NumFECPackets(0),
	)
	if err != nil {
		panic(err)
	}
	reg.Add(fecFactory)

	i, err := reg.Build("")
	if err != nil {
		panic(err)
	}
	defer func() { _ = i.Close() }()

	var fecOut atomic.Int64
	fecSink := interceptor.RTPWriterFunc(func(h *rtp.Header, payload []byte, a interceptor.Attributes) (int, error) {
		if h.PayloadType == fecPT && h.SSRC == fecSSRC {
			fecOut.Add(1)
		}
		return len(payload), nil
	})

	streamInfo := &interceptor.StreamInfo{
		ID:                                "media",
		SSRC:                              mediaSSRC,
		PayloadType:                       mediaPT,
		SSRCForwardErrorCorrection:        fecSSRC,
		PayloadTypeForwardErrorCorrection: fecPT,
	}

	mediaWriter := i.BindLocalStream(streamInfo, fecSink)

	// 1) FEC disabled
	bus.Publish(mediaSSRC, flexfec.RuntimeConfig{
		Enabled:         false,
		NumMediaPackets: 10,
		NumFECPackets:   0,
	})

	_ = sendMediaN(mediaWriter, mediaSSRC, mediaPT, 1000, 123456, 30)
	time.Sleep(50 * time.Millisecond)
	before := fecOut.Load()

	// 2) Enable FEC: k=10, r=2
	bus.Publish(mediaSSRC, flexfec.RuntimeConfig{
		Enabled:         true,
		NumMediaPackets: 10,
		NumFECPackets:   2,
	})

	_ = sendMediaN(mediaWriter, mediaSSRC, mediaPT, 2000, 223456, 30)
	time.Sleep(50 * time.Millisecond)
	after := fecOut.Load()

	fmt.Printf("fec packets before enable: %d\n", before)
	fmt.Printf("fec packets after enable:  %d\n", after)
	if after <= before {
		panic("expected FEC output to increase after enabling (k,r)")
	}
	fmt.Println("OK: runtime update changed fec output")
}

func sendMediaN(w interceptor.RTPWriter, ssrc uint32, pt uint8, startSeq uint16, startTS uint32, n int) error {
	seq := startSeq
	ts := startTS

	payload := []byte{0x01, 0x02, 0x03, 0x04} // dummy payload

	for i := 0; i < n; i++ {
		h := &rtp.Header{
			Version:        2,
			PayloadType:    pt,
			SequenceNumber: seq,
			Timestamp:      ts,
			SSRC:           ssrc,
		}
		if _, err := w.Write(h, payload, nil); err != nil {
			return err
		}
		seq++
		ts += 3000 // dummy increment (z.B. 90kHz/30fps)
	}
	return nil
}
