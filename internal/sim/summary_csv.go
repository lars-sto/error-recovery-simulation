package sim

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
)

type SummaryRow struct {
	Scenario string
	Mode     Mode
	Seed     int64

	DurationMs int64

	FinalLossDeadline   float64
	FinalLossNoDeadline float64

	OverheadRatioBytes float64
	OverheadRatioPkts  float64

	MeanQueueDelayMs float64

	MeanPolicyR        float64
	MaxPolicyR         uint32
	MeanPolicyOverhead float64

	MeanLossWindow float64
	MaxLossWindow  float64

	SentMediaPkts int64
	SentFECPkts   int64
	DroppedMedia  int64
	DroppedFEC    int64
	QueueDrops    int64
	WireDrops     int64

	RecoveredPkts      int64
	UniquePkts         int64
	GoodWithinDeadline int64
}

type SummaryCSVWriter struct {
	f *os.File
	w *csv.Writer
}

func NewSummaryCSVWriter(path string) (*SummaryCSVWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := csv.NewWriter(f)

	hdr := []string{
		"scenario",
		"mode",
		"seed",
		"duration_ms",
		"final_loss_deadline",
		"final_loss_no_deadline",
		"overhead_ratio_bytes",
		"overhead_ratio_pkts",
		"mean_queue_delay_ms",
		"mean_policy_r",
		"max_policy_r",
		"mean_policy_overhead",
		"mean_loss_window",
		"max_loss_window",
		"sent_media_pkts",
		"sent_fec_pkts",
		"dropped_media_pkts",
		"dropped_fec_pkts",
		"queue_drops_pkts",
		"wire_drops_pkts",
		"recovered_pkts",
		"unique_pkts",
		"good_within_deadline",
	}
	if err := w.Write(hdr); err != nil {
		_ = f.Close()
		return nil, err
	}
	w.Flush()
	return &SummaryCSVWriter{f: f, w: w}, nil
}

func (s *SummaryCSVWriter) WriteRow(r SummaryRow) error {
	row := []string{
		r.Scenario,
		string(r.Mode),
		strconv.FormatInt(r.Seed, 10),
		strconv.FormatInt(r.DurationMs, 10),

		ff(r.FinalLossDeadline),
		ff(r.FinalLossNoDeadline),

		ff(r.OverheadRatioBytes),
		ff(r.OverheadRatioPkts),

		ff(r.MeanQueueDelayMs),

		ff(r.MeanPolicyR),
		strconv.FormatUint(uint64(r.MaxPolicyR), 10),
		ff(r.MeanPolicyOverhead),

		ff(r.MeanLossWindow),
		ff(r.MaxLossWindow),

		strconv.FormatInt(r.SentMediaPkts, 10),
		strconv.FormatInt(r.SentFECPkts, 10),
		strconv.FormatInt(r.DroppedMedia, 10),
		strconv.FormatInt(r.DroppedFEC, 10),
		strconv.FormatInt(r.QueueDrops, 10),
		strconv.FormatInt(r.WireDrops, 10),
		strconv.FormatInt(r.RecoveredPkts, 10),
		strconv.FormatInt(r.UniquePkts, 10),
		strconv.FormatInt(r.GoodWithinDeadline, 10),
	}
	return s.w.Write(row)
}

func (s *SummaryCSVWriter) Close() error {
	s.w.Flush()
	if err := s.w.Error(); err != nil {
		_ = s.f.Close()
		return err
	}
	return s.f.Close()
}
