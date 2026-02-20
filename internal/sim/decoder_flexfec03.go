// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT
// modification of https://github.com/pion/interceptor/blob/master/pkg/flexfec/flexfec_decoder_03.go

package sim

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/pion/logging"
	"github.com/pion/rtp"
)

var (
	errPacketTruncated                = errors.New("packet truncated")
	errRetransmissionBitSet           = errors.New("packet with retransmission bit set not supported")
	errInflexibleGeneratorMatrix      = errors.New("packet with inflexible generator matrix not supported")
	errMultipleSSRCProtection         = errors.New("multiple ssrc protection not supported")
	errLastOptionalMaskKBitSetToFalse = errors.New("k-bit of last optional mask is set to false")
)

// FlexFEC03Decoder is a minimal FlexFEC-03 decoder used for simulation
// It attempts recovery when exactly one protected media packet is missing
type FlexFEC03Decoder struct {
	logger              logging.LeveledLogger
	ssrc                uint32
	protectedStreamSSRC uint32

	maxMediaPackets int
	maxFECPackets   int

	recoveredPackets   []rtp.Packet
	receivedFECPackets []fecPacketState
}

func NewFlexFEC03Decoder(ssrc uint32, protectedStreamSSRC uint32) *FlexFEC03Decoder {
	return &FlexFEC03Decoder{
		logger:              logging.NewDefaultLoggerFactory().NewLogger("fec_decoder"),
		ssrc:                ssrc,
		protectedStreamSSRC: protectedStreamSSRC,
		maxMediaPackets:     200,
		maxFECPackets:       200,
		recoveredPackets:    make([]rtp.Packet, 0),
		receivedFECPackets:  make([]fecPacketState, 0),
	}
}

// Push inserts a packet (media or fec) and returns newly recovered media packets (if any)
func (d *FlexFEC03Decoder) Push(receivedPacket rtp.Packet) []rtp.Packet {
	// Buffer reset heuristic on large gaps (media stream only)
	if len(d.recoveredPackets) == d.maxMediaPackets {
		back := d.recoveredPackets[len(d.recoveredPackets)-1]
		if back.SSRC == receivedPacket.SSRC {
			seqDiffVal := seqDiff(receivedPacket.SequenceNumber, back.SequenceNumber)
			if seqDiffVal > uint16(d.maxMediaPackets) {
				d.logger.Info("big gap in media sequence numbers - resetting buffers")
				d.recoveredPackets = nil
				d.receivedFECPackets = nil
			}
		}
	}

	d.insertPacket(receivedPacket)
	return d.attemptRecovery()
}

func (d *FlexFEC03Decoder) insertPacket(receivedPkt rtp.Packet) {
	// Remove very old FEC packets (by sequence distance to newest incoming FEC)
	if len(d.receivedFECPackets) > 0 && receivedPkt.SSRC == d.ssrc {
		toRemove := 0
		for _, fecPkt := range d.receivedFECPackets {
			if abs(int(receivedPkt.SequenceNumber)-int(fecPkt.packet.SequenceNumber)) > 0x3fff {
				toRemove++
			} else {
				break
			}
		}
		if toRemove > 0 {
			d.receivedFECPackets = d.receivedFECPackets[toRemove:]
		}
	}

	switch receivedPkt.SSRC {
	case d.ssrc:
		d.insertFECPacket(receivedPkt)
	case d.protectedStreamSSRC:
		d.insertMediaPacket(receivedPkt)
	default:
		// ignore
	}

	d.discardOldRecoveredPackets()
}

func (d *FlexFEC03Decoder) insertMediaPacket(receivedPkt rtp.Packet) {
	for _, rp := range d.recoveredPackets {
		if rp.SequenceNumber == receivedPkt.SequenceNumber {
			return
		}
	}

	d.recoveredPackets = append(d.recoveredPackets, receivedPkt)
	sort.Slice(d.recoveredPackets, func(i, j int) bool {
		return isNewerSeq(d.recoveredPackets[i].SequenceNumber, d.recoveredPackets[j].SequenceNumber)
	})
	d.updateCoveringFecPackets(receivedPkt)
}

func (d *FlexFEC03Decoder) updateCoveringFecPackets(receivedPkt rtp.Packet) {
	for i := range d.receivedFECPackets {
		fecPkt := &d.receivedFECPackets[i]
		for _, protectedPacket := range fecPkt.protectedPackets {
			if protectedPacket.seq == receivedPkt.SequenceNumber {
				protectedPacket.packet = &receivedPkt
			}
		}
	}
}

func (d *FlexFEC03Decoder) insertFECPacket(fecPkt rtp.Packet) { //nolint:cyclop
	for _, existing := range d.receivedFECPackets {
		if existing.packet.SequenceNumber == fecPkt.SequenceNumber {
			return
		}
	}

	fec, err := parseFlexFEC03Header(fecPkt.Payload)
	if err != nil {
		d.logger.Errorf("failed to parse flexfec03 header: %v", err)

		return
	}

	if fec.protectedSSRC != d.protectedStreamSSRC {
		d.logger.Errorf("fec protects unknown ssrc, expected %d, got %d", d.protectedStreamSSRC, fec.protectedSSRC)

		return
	}

	protectedSeqs := decodeMask(uint64(fec.mask0), 15, fec.seqNumBase)
	if fec.mask1 != 0 {
		protectedSeqs = append(protectedSeqs, decodeMask(uint64(fec.mask1), 31, fec.seqNumBase+15)...)
	}
	if fec.mask2 != 0 {
		protectedSeqs = append(protectedSeqs, decodeMask(fec.mask2, 63, fec.seqNumBase+46)...)
	}

	if len(protectedSeqs) == 0 {
		d.logger.Warn("empty fec packet mask")

		return
	}

	protectedPackets := make([]*protectedPacket, 0, len(protectedSeqs))
	protectedSeqIt := 0
	recoveredIt := 0

	for protectedSeqIt < len(protectedSeqs) && recoveredIt < len(d.recoveredPackets) {
		switch {
		case isNewerSeq(protectedSeqs[protectedSeqIt], d.recoveredPackets[recoveredIt].SequenceNumber):
			protectedPackets = append(protectedPackets, &protectedPacket{seq: protectedSeqs[protectedSeqIt], packet: nil})
			protectedSeqIt++
		case isNewerSeq(d.recoveredPackets[recoveredIt].SequenceNumber, protectedSeqs[protectedSeqIt]):
			recoveredIt++
		default:
			protectedPackets = append(protectedPackets, &protectedPacket{seq: protectedSeqs[protectedSeqIt], packet: &d.recoveredPackets[recoveredIt]})
			protectedSeqIt++
			recoveredIt++
		}
	}

	for protectedSeqIt < len(protectedSeqs) {
		protectedPackets = append(protectedPackets, &protectedPacket{seq: protectedSeqs[protectedSeqIt], packet: nil})
		protectedSeqIt++
	}

	d.receivedFECPackets = append(d.receivedFECPackets, fecPacketState{
		packet:           fecPkt,
		flexFec:          fec,
		protectedPackets: protectedPackets,
	})

	sort.Slice(d.receivedFECPackets, func(i, j int) bool {
		return isNewerSeq(d.receivedFECPackets[i].packet.SequenceNumber, d.receivedFECPackets[j].packet.SequenceNumber)
	})

	if len(d.receivedFECPackets) > d.maxFECPackets {
		d.receivedFECPackets = d.receivedFECPackets[1:]
	}
}

func (d *FlexFEC03Decoder) attemptRecovery() []rtp.Packet {
	recoveredPackets := make([]rtp.Packet, 0)

	for {
		packetsRecovered := 0

		for i := range d.receivedFECPackets {
			fecPkt := &d.receivedFECPackets[i]

			missing := 0
			for _, pkt := range fecPkt.protectedPackets {
				if pkt.packet == nil {
					missing++
					if missing > 1 {
						break
					}
				}
			}
			if missing != 1 {
				continue
			}

			recovered, err := d.recoverPacket(fecPkt)
			if err != nil {
				d.logger.Errorf("failed to recover packet: %v", err)
				continue
			}

			recoveredPackets = append(recoveredPackets, recovered)
			d.recoveredPackets = append(d.recoveredPackets, recovered)
			sort.Slice(d.recoveredPackets, func(i, j int) bool {
				return isNewerSeq(d.recoveredPackets[i].SequenceNumber, d.recoveredPackets[j].SequenceNumber)
			})

			d.updateCoveringFecPackets(recovered)
			d.discardOldRecoveredPackets()

			packetsRecovered++
		}

		if packetsRecovered == 0 {
			break
		}
	}

	return recoveredPackets
}

func (d *FlexFEC03Decoder) recoverPacket(fec *fecPacketState) (rtp.Packet, error) {
	headerRecovery := make([]byte, 12)
	copy(headerRecovery, fec.packet.Payload[:10])

	var missingSeq uint16

	for _, protected := range fec.protectedPackets {
		if protected.packet != nil {
			receivedHeader, err := protected.packet.Header.Marshal()
			if err != nil {
				return rtp.Packet{}, fmt.Errorf("marshal received header: %w", err)
			}
			binary.BigEndian.PutUint16(receivedHeader[2:4], uint16(protected.packet.MarshalSize()-12))
			for i := 0; i < 8; i++ {
				headerRecovery[i] ^= receivedHeader[i]
			}
		} else {
			missingSeq = protected.seq
		}
	}

	headerRecovery[0] |= 0x80 // V=2
	headerRecovery[0] &= 0xbf // clear padding bit
	payloadLength := binary.BigEndian.Uint16(headerRecovery[2:4])

	// Recover missing sequence number
	binary.BigEndian.PutUint16(headerRecovery[2:4], missingSeq)
	// Recover SSRC for protected stream
	binary.BigEndian.PutUint32(headerRecovery[8:12], d.protectedStreamSSRC)

	payloadRecovery := make([]byte, payloadLength)
	copy(payloadRecovery, fec.flexFec.payload)

	for _, protected := range fec.protectedPackets {
		if protected.packet == nil {
			continue
		}
		raw, err := protected.packet.Marshal()
		if err != nil {
			return rtp.Packet{}, fmt.Errorf("marshal protected packet: %w", err)
		}
		// XOR protected payloads into recovery payload
		for i := 0; i < min(int(payloadLength), len(raw)-12); i++ {
			payloadRecovery[i] ^= raw[12+i]
		}
	}

	rawRecovered := append(headerRecovery, payloadRecovery...) //nolint:makezero

	var pkt rtp.Packet
	if err := pkt.Unmarshal(rawRecovered); err != nil {
		return rtp.Packet{}, fmt.Errorf("unmarshal recovered: %w", err)
	}
	return pkt, nil
}

func (d *FlexFEC03Decoder) discardOldRecoveredPackets() {
	const limit = 256
	if len(d.recoveredPackets) > limit {
		d.recoveredPackets = d.recoveredPackets[len(d.recoveredPackets)-limit:]
	}
}

func decodeMask(mask uint64, bitCount uint16, seqNumBase uint16) []uint16 {
	res := make([]uint16, 0)
	for i := uint16(0); i < bitCount; i++ {
		if (mask>>(bitCount-1-i))&1 == 1 {
			res = append(res, seqNumBase+i)
		}
	}
	return res
}

type fecPacketState struct {
	packet           rtp.Packet
	flexFec          flexFec
	protectedPackets []*protectedPacket
}

type flexFec struct {
	protectedSSRC uint32
	seqNumBase    uint16
	mask0         uint16
	mask1         uint32
	mask2         uint64
	payload       []byte
}

type protectedPacket struct {
	seq    uint16
	packet *rtp.Packet
}

func parseFlexFEC03Header(data []byte) (flexFec, error) {
	if len(data) < 20 {
		return flexFec{}, fmt.Errorf("%w: length %d", errPacketTruncated, len(data))
	}

	rBit := (data[0] & 0x80) != 0
	if rBit {
		return flexFec{}, errRetransmissionBitSet
	}

	fBit := (data[0] & 0x40) != 0
	if fBit {
		return flexFec{}, errInflexibleGeneratorMatrix
	}

	ssrcCount := data[8]
	if ssrcCount != 1 {
		return flexFec{}, fmt.Errorf("%w: count %d", errMultipleSSRCProtection, ssrcCount)
	}

	protectedSSRC := binary.BigEndian.Uint32(data[12:])
	seqNumBase := binary.BigEndian.Uint16(data[16:])

	rawPacketMask := data[18:]
	var payload []byte

	kBit0 := (rawPacketMask[0] & 0x80) != 0
	maskPart0 := binary.BigEndian.Uint16(rawPacketMask[0:2]) & 0x7FFF
	var maskPart1 uint32
	var maskPart2 uint64

	if kBit0 {
		payload = rawPacketMask[2:]
	} else {
		if len(data) < 24 {
			return flexFec{}, fmt.Errorf("%w: length %d", errPacketTruncated, len(data))
		}

		kBit1 := (rawPacketMask[2] & 0x80) != 0
		maskPart1 = binary.BigEndian.Uint32(rawPacketMask[2:]) & 0x7FFFFFFF

		if kBit1 {
			payload = rawPacketMask[6:]
		} else {
			if len(data) < 32 {
				return flexFec{}, fmt.Errorf("%w: length %d", errPacketTruncated, len(data))
			}

			kBit2 := (rawPacketMask[6] & 0x80) != 0
			maskPart2 = binary.BigEndian.Uint64(rawPacketMask[6:]) & 0x7FFFFFFFFFFFFFFF

			if kBit2 {
				payload = rawPacketMask[14:]
			} else {
				return flexFec{}, errLastOptionalMaskKBitSetToFalse
			}
		}
	}

	return flexFec{
		protectedSSRC: protectedSSRC,
		seqNumBase:    seqNumBase,
		mask0:         maskPart0,
		mask1:         maskPart1,
		mask2:         maskPart2,
		payload:       payload,
	}, nil
}

func seqDiff(a, b uint16) uint16 {
	return min(a-b, b-a)
}

func abs(x int) int {
	if x >= 0 {
		return x
	}
	return -x
}

func isNewerSeq(prevValue, value uint16) bool {
	breakpoint := uint16(0x8000)
	if value-prevValue == breakpoint {
		return value > prevValue
	}
	return value != prevValue && (value-prevValue) < breakpoint
}

func min[T ~int | ~uint16](a, b T) T {
	if a < b {
		return a
	}
	return b
}
