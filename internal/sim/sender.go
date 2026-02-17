package sim

import (
	"context"
	"time"

	"github.com/pion/rtp"
)

func (e *Env) RunSender(ctx context.Context) error {
	ticker := time.NewTicker(time.Second / time.Duration(e.sc.PacketRateHz))
	defer ticker.Stop()

	var seq uint16 = 1
	var ts uint32 = 1

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			h := &rtp.Header{
				SSRC:           e.sc.SSRCMedia,
				PayloadType:    e.sc.PTMedia,
				SequenceNumber: seq,
				Timestamp:      ts,
			}
			payload := make([]byte, e.sc.PayloadSize)
			// optional: deterministic pattern
			if _, err := e.writeA.Write(h, payload, nil); err != nil {
				return err
			}

			seq++
			ts += 3000 // arbitrary clock step for synthetic
		}
	}
}
