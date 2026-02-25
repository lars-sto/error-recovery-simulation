package sim

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type TimeSample struct {
	T time.Duration

	LossWindow float64
	TargetBWE  float64
	MediaRate  float64

	CapacityBps       float64
	CurrentBitrateBps float64
	QueueDelayMs      float64

	PolicyEnabled  bool
	PolicyK        uint32
	PolicyR        uint32
	PolicyOverhead float64

	SentMedia    int64
	SentFEC      int64
	DroppedMedia int64
	DroppedFEC   int64
	QueueDrops   int64
	WireDrops    int64
}

type Recorder interface {
	OnSample(s TimeSample)
	Close() error
}

type CSVRecorder struct {
	f *os.File
	w *csv.Writer
}

func NewCSVRecorder(path string) (*CSVRecorder, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := csv.NewWriter(f)

	hdr := []string{
		"t_ms",
		"loss_window",
		"target_bwe_bps",
		"media_rate_bps",
		"capacity_bps",
		"current_bitrate_bps",
		"queue_delay_ms",
		"policy_enabled",
		"policy_k",
		"policy_r",
		"policy_overhead",
		"sent_media",
		"sent_fec",
		"dropped_media",
		"dropped_fec",
		"queue_drops",
		"wire_drops",
	}
	if err := w.Write(hdr); err != nil {
		_ = f.Close()
		return nil, err
	}
	w.Flush()

	return &CSVRecorder{f: f, w: w}, nil
}

func (r *CSVRecorder) OnSample(s TimeSample) {
	row := []string{
		strconv.FormatInt(s.T.Milliseconds(), 10),
		ff(s.LossWindow),
		ff(s.TargetBWE),
		ff(s.MediaRate),
		ff(s.CapacityBps),
		ff(s.CurrentBitrateBps),
		ff(s.QueueDelayMs),
		strconv.FormatBool(s.PolicyEnabled),
		strconv.FormatUint(uint64(s.PolicyK), 10),
		strconv.FormatUint(uint64(s.PolicyR), 10),
		ff(s.PolicyOverhead),
		strconv.FormatInt(s.SentMedia, 10),
		strconv.FormatInt(s.SentFEC, 10),
		strconv.FormatInt(s.DroppedMedia, 10),
		strconv.FormatInt(s.DroppedFEC, 10),
		strconv.FormatInt(s.QueueDrops, 10),
		strconv.FormatInt(s.WireDrops, 10),
	}
	_ = r.w.Write(row)
}

func (r *CSVRecorder) Close() error {
	r.w.Flush()
	if err := r.w.Error(); err != nil {
		_ = r.f.Close()
		return err
	}
	return r.f.Close()
}

func ff(v float64) string { return fmt.Sprintf("%.6f", v) }
