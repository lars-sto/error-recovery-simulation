package sim

import (
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/flexfec"
	"github.com/pion/rtp"
	"github.com/pion/transport/v4/vnet"
)

type Env struct {
	sc   Scenario
	mode Mode
	log  *CSVLogger
	rng  *rand.Rand

	// vnet
	net   *vnet.Router
	wan   *vnet.Net
	lanA  *vnet.Net
	lanB  *vnet.Net
	close func() error

	// interceptor
	regA *interceptor.Registry
	regB *interceptor.Registry

	// writers/readers
	writeA interceptor.RTPWriter // sender pipeline out
	readB  interceptor.RTPReader // receiver pipeline in

	// seq tracking
	sentMedia atomic.Int64
	sentFEC   atomic.Int64
	recvMedia atomic.Int64
	recvFEC   atomic.Int64
}

func NewEnv(sc Scenario, mode Mode, logger *CSVLogger) (*Env, error) {
	e := &Env{
		sc:   sc,
		mode: mode,
		log:  logger,
		rng:  rand.New(rand.NewSource(sc.Seed)),
	}

	if err := e.setupVNet(); err != nil {
		return nil, err
	}
	if err := e.setupInterceptors(); err != nil {
		_ = e.Close()
		return nil, err
	}
	return e, nil
}

func (e *Env) Close() error {
	if e.close != nil {
		return e.close()
	}
	return nil
}

func (e *Env) setupVNet() error {
	// Simple router topology:
	// hostA -- lanA -- router -- wan(loss) -- router -- lanB -- hostB
	// In vnet we typically configure one Router with multiple Nets.

	router, err := vnet.NewRouter(&vnet.RouterConfig{
		CIDR: "1.0.0.0/8",
	})
	if err != nil {
		return err
	}

	lanA, err := vnet.NewNet(&vnet.NetConfig{StaticIPs: []string{"1.1.1.1"}})
	if err != nil {
		return err
	}
	lanB, err := vnet.NewNet(&vnet.NetConfig{StaticIPs: []string{"1.1.1.1"}})
	if err != nil {
		return err
	}

	// “wan” net: apply random loss here
	wan, err := vnet.NewNet(&vnet.NetConfig{
		StaticIPs: []string{"1.1.1.254"},
	})
	if err != nil {
		return err
	}

	if err := router.AddNet(lanA); err != nil {
		return err
	}
	if err := router.AddNet(lanB); err != nil {
		return err
	}
	if err := router.AddNet(wan); err != nil {
		return err
	}

	if err := router.Start(); err != nil {
		return err
	}

	e.net = router
	e.lanA = lanA
	e.lanB = lanB
	e.wan = wan
	e.close = func() error {
		return router.Stop()
	}
	return nil
}

func (e *Env) setupInterceptors() error {
	// Sender side registry (A)
	e.regA = &interceptor.Registry{}

	// Receiver side registry (B)
	e.regB = &interceptor.Registry{}

	// FlexFEC interceptor: for now, same on both ends.
	// For ModeAdaptive: später runtime config source vom Engine->Bus.
	fecFactory, err := flexfec.NewFecInterceptor(
		flexfec.NumMediaPackets(10),
		flexfec.NumFECPackets(2),
	)
	if err != nil {
		return err
	}

	e.regA.Add(fecFactory)
	e.regB.Add(fecFactory)

	// Build stacks
	iA, err := e.regA.Build("")
	if err != nil {
		return err
	}
	iB, err := e.regB.Build("")
	if err != nil {
		_ = iA.Close()
		return err
	}

	// Underlying transport endpoints are attached later; here we just keep interceptor nodes.
	_ = iA
	_ = iB

	// We keep it simple for the scaffold:
	// - Sender writes RTP into interceptor chain -> we capture output RTP/FEC via RTPWriterFunc
	// - Receiver reads RTP from “network” -> pass into interceptor chain -> capture output via RTPReaderFunc
	//
	// The next step (after this scaffold) is to wire actual pion/webrtc PeerConnections over vnet.
	// For now we simulate “wire” using channels.

	// Create “wire” channels between A and B:
	wire := make(chan rtp.Packet, 2048)

	// Sender: top of chain writes into wire
	e.writeA = interceptor.RTPWriterFunc(func(h *rtp.Header, payload []byte, _ interceptor.Attributes) (int, error) {
		pkt := rtp.Packet{Header: *h, Payload: append([]byte(nil), payload...)}
		kind := "media"
		if h.SSRC == e.sc.SSRCFEC || h.PayloadType == e.sc.PTFEC {
			kind = "fec"
			e.sentFEC.Add(1)
		} else {
			e.sentMedia.Add(1)
		}
		e.log.Packet("tx", kind, time.Now(), h.SSRC, h.PayloadType, h.SequenceNumber, h.Timestamp, len(payload), false, false)

		// Apply “network” loss ourselves (scaffold-level).
		// When we move to full pion+vnet transport, this goes away.
		if e.rng.Float64() < e.sc.Network.RandomLoss {
			return len(payload), nil
		}

		wire <- pkt
		return len(payload), nil
	})

	// Receiver: reads from wire
	e.readB = interceptor.RTPReaderFunc(func(_ interceptor.Attributes) (*rtp.Packet, interceptor.Attributes, error) {
		pkt, ok := <-wire
		if !ok {
			return nil, nil, ioEOF{}
		}
		return &pkt, nil, nil
	})

	// Attach iA/iB:
	// - In real pipeline you would bind interceptor streams to SSRC/PT.
	// For scaffold, we directly use e.writeA/e.readB and keep the rest for later.
	_ = iA
	_ = iB

	return nil
}

type ioEOF struct{}

func (ioEOF) Error() string { return "eof" }
