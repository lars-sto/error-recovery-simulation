// internal/sim/runner.go
package sim

import (
	"context"
	"math"
	"runtime"
	"time"

	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
	"github.com/lars-sto/error-recovery-simulation/internal/adapter"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/flexfec"
	"github.com/pion/rtp"
)

type RunOptions struct {
	Mode     Mode
	Seed     int64
	Recorder Recorder
}

type simStatsSource struct {
	ch chan recovery.NetworkStats
}

func newSimStatsSource() *simStatsSource {
	// unbuffered helps ordering; we additionally wait for observer-ack
	return &simStatsSource{ch: make(chan recovery.NetworkStats)}
}

func (s *simStatsSource) Stats() <-chan recovery.NetworkStats { return s.ch }
func (s *simStatsSource) Close()                              { close(s.ch) }

type simObserver struct {
	processed chan struct{}
}

func (o *simObserver) OnSample(_ recovery.NetworkStats, _ recovery.PolicyDecision, _ bool) {
	// signal "engine processed one stats sample"
	select {
	case o.processed <- struct{}{}:
	default:
		// never block
	}
}

func RunScenario(sc Scenario, opt RunOptions) (RunResult, error) {
	res := RunResult{
		Scenario: sc.Name,
		Mode:     opt.Mode,
		Seed:     opt.Seed,
		Duration: sc.Duration,
	}

	// Seed link jitter + loss model deterministically per run
	linkSpec := sc.Link
	linkSpec.Seed = opt.Seed
	linkSpec.Loss = reseedLossModel(sc.Link.Loss, opt.Seed)

	start := sc.Sender.StartTime
	if start.IsZero() {
		start = time.Unix(0, 0)
	}
	end := start.Add(sc.Duration)

	link := NewLink(linkSpec, start)
	recv := NewReceiver(sc.IDs)

	// Maps for deadline metric
	sendAt := make(map[uint16]time.Time, int(sc.Duration/sc.Sender.Interval())+8)

	// Pion interceptor stack (FlexFEC encoder)
	bus := adapter.NewRuntimeBus()
	flexAdapter := adapter.NewFlexFECAdapter(bus)

	reg := &interceptor.Registry{}

	initialR := sc.StaticR
	fecFactory, err := flexfec.NewFecInterceptor(
		flexfec.WithConfigSource(bus),
		flexfec.NumMediaPackets(sc.K),
		flexfec.NumFECPackets(initialR),
	)
	if err != nil {
		return res, err
	}
	reg.Add(fecFactory)

	i, err := reg.Build("")
	if err != nil {
		return res, err
	}
	defer func() { _ = i.Close() }()

	// Track "current policy" for recorder. (Static: fixed, Adaptive: updated when engine publishes)
	polEnabled := (initialR > 0)
	polK := sc.K
	polR := initialR
	polOver := overhead(polK, polR)

	// Writer at end of pipeline: push packets into Link with current virtual send-time
	var now time.Time

	var (
		sentMediaPkts  int64
		sentFECPkts    int64
		sentMediaBytes int64
		sentFECBytes   int64

		droppedMediaPkts int64
		droppedFECPkts   int64
		droppedQueuePkts int64
		droppedWirePkts  int64
	)

	// Per-stats-window deltas (media-focused for loss rate)
	var winSentMedia int64
	var winDropMedia int64
	var winBytesTotal int64

	linkWriter := interceptor.RTPWriterFunc(func(h *rtp.Header, payload []byte, _ interceptor.Attributes) (int, error) {
		p := make([]byte, len(payload))
		copy(p, payload)

		pkt := rtp.Packet{Header: *h, Payload: p}

		isFEC := (h.SSRC == sc.IDs.FECSSRC) || (h.PayloadType == sc.IDs.FECPT)
		out := link.Send(pkt, now, isFEC)

		if isFEC {
			sentFECPkts++
			sentFECBytes += int64(out.SizeBytes)
		} else {
			sentMediaPkts++
			sentMediaBytes += int64(out.SizeBytes)

			winSentMedia++
		}
		winBytesTotal += int64(out.SizeBytes)

		if out.Dropped {
			if isFEC {
				droppedFECPkts++
			} else {
				droppedMediaPkts++
				winDropMedia++
			}
			switch out.Reason {
			case DropQueue:
				droppedQueuePkts++
			case DropWireLoss, DropZeroCap:
				droppedWirePkts++
			}
		}

		return len(payload), nil
	})

	streamInfo := &interceptor.StreamInfo{
		SSRC:                              sc.IDs.MediaSSRC,
		PayloadType:                       sc.IDs.MediaPT,
		SSRCForwardErrorCorrection:        sc.IDs.FECSSRC,
		PayloadTypeForwardErrorCorrection: sc.IDs.FECPT,
	}
	pipelineWriter := i.BindLocalStream(streamInfo, linkWriter)
	defer i.UnbindLocalStream(streamInfo)

	// Adaptive engine (optional)
	var (
		statsSrc  *simStatsSource
		observer  *simObserver
		engineCtx context.Context
		cancel    context.CancelFunc
	)
	if opt.Mode == ModeAdaptive {
		statsSrc = newSimStatsSource()
		observer = &simObserver{processed: make(chan struct{}, 16)}

		sink := adapter.SinkFunc(func(d recovery.PolicyDecision) {
			flexAdapter.Apply(sc.IDs.MediaSSRC, d)

			// update policy snapshot for recorder
			f := d.FEC
			polEnabled = f.Enabled
			polK = f.NumMediaPackets
			polR = f.NumFECPackets
			polOver = overhead(polK, polR)
		})

		engineCfg := recovery.DefaultConfig()
		engineCfg.Scheme = recovery.FECSchemeFlexFEC03

		engine := recovery.NewEngine(engineCfg, statsSrc, sink, observer)
		engineCtx, cancel = context.WithCancel(context.Background())
		go engine.Run(engineCtx)
	}

	// Event times
	interval := sc.Sender.Interval()
	if interval <= 0 {
		interval = 20 * time.Millisecond
	}
	statsEvery := sc.StatsInterval
	if statsEvery <= 0 {
		statsEvery = 200 * time.Millisecond
	}

	nextMedia := start
	nextStats := start.Add(statsEvery)

	// Main event loop: process next (delivery | stats | media) in time order
	for {
		tDel, hasDel := peekDelivery(link)

		mediaEnabled := nextMedia.Before(end) || nextMedia.Equal(end)
		statsEnabled := nextStats.Before(end) || nextStats.Equal(end)

		next := time.Time{}
		set := false

		if hasDel {
			next = tDel
			set = true
		}
		if statsEnabled && (!set || nextStats.Before(next)) {
			next = nextStats
			set = true
		}
		if mediaEnabled && (!set || nextMedia.Before(next)) {
			next = nextMedia
			set = true
		}

		if !set {
			break
		}

		now = next

		// Priority: deliver first if equal time, then stats, then media
		if hasDel && now.Equal(tDel) {
			dp, _ := link.Next()
			recv.OnPacket(dp.Pkt, dp.Arrives)
			continue
		}

		if statsEnabled && now.Equal(nextStats) {
			elapsed := now.Sub(start)

			// Loss window (pre-FEC media drop ratio)
			loss := 0.0
			if winSentMedia > 0 {
				loss = float64(winDropMedia) / float64(winSentMedia)
				loss = clamp01(loss)
			}

			// BWE schedule given to engine as TargetBitrate
			targetBWE := 0.0
			if sc.BWE != nil {
				targetBWE = sc.BWE.At(elapsed)
			}

			// Current bitrate: bytes sent in window / window duration
			winSec := statsEvery.Seconds()
			currentBps := 0.0
			if winSec > 0 {
				currentBps = float64(winBytesTotal*8) / winSec
			}

			// Push to engine, wait for observer ack (engine processed)
			if opt.Mode == ModeAdaptive && statsSrc != nil && observer != nil {
				statsSrc.ch <- recovery.NetworkStats{
					RTTMs:          sc.RTTMs,
					JitterMs:       sc.JitterMs,
					LossRate:       loss,
					TargetBitrate:  targetBWE,
					CurrentBitrate: currentBps,
					Timestamp:      now,
				}
				// Ensure engine has consumed this stats sample (even if it didn't publish a new decision)
				<-observer.processed
			}

			// Recorder sample (always)
			if opt.Recorder != nil {
				opt.Recorder.OnSample(TimeSample{
					T:              elapsed,
					LossWindow:     loss,
					TargetBWE:      targetBWE,
					MediaRate:      sc.Sender.MediaBitrateBps(true),
					PolicyEnabled:  polEnabled,
					PolicyK:        polK,
					PolicyR:        polR,
					PolicyOverhead: polOver,
					SentMedia:      sentMediaPkts,
					SentFEC:        sentFECPkts,
					DroppedMedia:   droppedMediaPkts,
					DroppedFEC:     droppedFECPkts,
					QueueDrops:     droppedQueuePkts,
					WireDrops:      droppedWirePkts,
				})
			}

			// Reset window counters
			winSentMedia = 0
			winDropMedia = 0
			winBytesTotal = 0

			nextStats = nextStats.Add(statsEvery)
			continue
		}

		if mediaEnabled && now.Equal(nextMedia) {
			seq := sc.Sender.StartSeq + uint16(len(sendAt))
			ts := sc.Sender.StartTS + uint32(len(sendAt))*sc.Sender.TimestampStep

			h := &rtp.Header{
				Version:        2,
				PayloadType:    sc.IDs.MediaPT,
				SequenceNumber: seq,
				Timestamp:      ts,
				SSRC:           sc.IDs.MediaSSRC,
			}

			payload := makePayload(opt.Seed, seq, sc.Sender.PayloadBytes)

			sendAt[seq] = now

			_, err := pipelineWriter.Write(h, payload, interceptor.Attributes{})
			if err != nil {
				return res, err
			}

			nextMedia = nextMedia.Add(interval)
			continue
		}
	}

	// Drain remaining deliveries after end (queue may push them beyond end)
	for {
		tDel, hasDel := peekDelivery(link)
		if !hasDel {
			break
		}
		now = tDel
		dp, _ := link.Next()
		recv.OnPacket(dp.Pkt, dp.Arrives)
	}

	// Stop engine + recorder
	if opt.Mode == ModeAdaptive && statsSrc != nil && cancel != nil {
		statsSrc.Close()
		cancel()
		// give engine goroutine a chance to exit cleanly
		runtime.Gosched()
	}
	if opt.Recorder != nil {
		_ = opt.Recorder.Close()
	}

	// Receiver snapshot
	snap := recv.Snapshot()

	res.SentMediaPkts = sentMediaPkts
	res.SentFECPkts = sentFECPkts
	res.SentMediaBytes = sentMediaBytes
	res.SentFECBytes = sentFECBytes

	res.DroppedMediaPkts = droppedMediaPkts
	res.DroppedFECPkts = droppedFECPkts
	res.DroppedQueuePkts = droppedQueuePkts
	res.DroppedWirePkts = droppedWirePkts

	res.RecvMediaPkts = snap.RecvMedia
	res.RecvFECPkts = snap.RecvFEC
	res.RecoveredPkts = snap.Recovered
	res.UniquePkts = snap.Unique

	if sentMediaPkts > 0 {
		res.OverheadRatioPkts = float64(sentFECPkts) / float64(sentMediaPkts)
		res.FinalLossNoDeadline = clamp01(1.0 - float64(snap.Unique)/float64(sentMediaPkts))
	}
	if sentMediaBytes > 0 {
		res.OverheadRatioBytes = float64(sentFECBytes) / float64(sentMediaBytes)
	}

	// Deadline-aware goodput: packet counts available by (sendAt + deadline)
	deadline := sc.PlayoutDeadline
	if deadline <= 0 {
		deadline = 200 * time.Millisecond
	}

	var good int64
	for seq, sAt := range sendAt {
		if aAt, ok := recv.availAt[seq]; ok {
			if !aAt.After(sAt.Add(deadline)) {
				good++
			}
		}
	}
	res.GoodWithinDeadline = good
	if sentMediaPkts > 0 {
		res.FinalLossDeadline = clamp01(1.0 - float64(good)/float64(sentMediaPkts))
	}

	return res, nil
}

func peekDelivery(l *Link) (time.Time, bool) {
	if l == nil || l.pq.Len() == 0 {
		return time.Time{}, false
	}
	return l.pq[0].at, true
}

func makePayload(seed int64, seq uint16, size int) []byte {
	if size <= 0 {
		return nil
	}
	out := make([]byte, size)
	// deterministic xorshift64*
	x := uint64(seed) ^ (uint64(seq) * 0x9e3779b97f4a7c15)
	if x == 0 {
		x = 0xdeadbeefcafebabe
	}
	for i := 0; i < size; i++ {
		x ^= x >> 12
		x ^= x << 25
		x ^= x >> 27
		y := x * 2685821657736338717
		out[i] = byte(y >> 56)
	}
	return out
}

func overhead(k, r uint32) float64 {
	if k == 0 {
		return 0
	}
	return float64(r) / float64(k)
}

func clamp01(x float64) float64 {
	return math.Max(0, math.Min(1, x))
}

func reseedLossModel(m LossModel, seed int64) LossModel {
	switch v := m.(type) {
	case *ScheduledBernoulliLoss:
		return NewScheduledBernoulliLoss(v.name, seed, v.P)
	case *GilbertElliottLoss:
		// re-create to reset per-SSRC states deterministically
		return NewGilbertElliottLoss(v.NameStr, seed, v.PGB, v.PBG, v.PG, v.PB)
	default:
		return m
	}
}
