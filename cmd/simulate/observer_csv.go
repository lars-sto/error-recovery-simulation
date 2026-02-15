package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
)

type CSVObserver struct {
	w *csv.Writer
	f *os.File
}

func NewCSVObserver(path string) (*CSVObserver, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	w := csv.NewWriter(f)
	if err := w.Write([]string{
		"time_ms",
		"loss",
		"rtt",
		"current_bitrate",
		"target_bitrate",
		"fec_enabled",
		"overhead",
		"reason",
		"changed",
	}); err != nil {
		_ = f.Close()
		return nil, err
	}

	return &CSVObserver{w: w, f: f}, nil
}

func (o *CSVObserver) Close() error {
	o.w.Flush()
	if err := o.w.Error(); err != nil {
		_ = o.f.Close()
		return err
	}
	return o.f.Close()
}

func (o *CSVObserver) OnSample(s recovery.NetworkStats, d recovery.PolicyDecision, changed bool) {
	_ = o.w.Write([]string{
		strconv.FormatInt(s.Timestamp.UnixMilli(), 10),
		fmtFloat(s.LossRate),
		strconv.Itoa(s.RTTMs),
		fmtFloat(s.CurrentBitrate),
		fmtFloat(s.TargetBitrate),
		strconv.FormatBool(d.FEC.Enabled),
		fmtFloat(d.FEC.TargetOverhead),
		d.FEC.Reason,
		strconv.FormatBool(changed),
	})
}
