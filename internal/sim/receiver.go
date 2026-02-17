package sim

import (
	"context"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

func (e *Env) RunReceiver(ctx context.Context) error {
	buf := make([]byte, 2048) // > MTU, reicht für RTP+Payload

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, _, err := e.readB.Read(buf, interceptor.Attributes{})
		if err != nil {
			return nil
		}

		var pkt rtp.Packet
		if err := pkt.Unmarshal(buf[:n]); err != nil {
			continue
		}

		kind := "media"
		if pkt.SSRC == e.sc.SSRCFEC || pkt.PayloadType == e.sc.PTFEC {
			kind = "fec"
			e.recvFEC.Add(1)
		} else {
			kind = "media"
			e.recvMedia.Add(1)
		}

		e.log.Packet("rx", kind, time.Now(),
			pkt.SSRC, pkt.PayloadType, pkt.SequenceNumber, pkt.Timestamp,
			len(pkt.Payload), false, false,
		)
	}
}
