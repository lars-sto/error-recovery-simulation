package sim

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type CSVLogger struct {
	mu sync.Mutex

	packetsF *os.File
	packetsW *csv.Writer

	summaryF *os.File
	summaryW *csv.Writer
}

func NewCSVLogger(outDir string) (*CSVLogger, error) {
	pf, err := os.Create(filepath.Join(outDir, "packets.csv"))
	if err != nil {
		return nil, err
	}
	sf, err := os.Create(filepath.Join(outDir, "summary.csv"))
	if err != nil {
		_ = pf.Close()
		return nil, err
	}

	l := &CSVLogger{
		packetsF: pf,
		packetsW: csv.NewWriter(pf),
		summaryF: sf,
		summaryW: csv.NewWriter(sf),
	}

	_ = l.packetsW.Write([]string{
		"t_unix_nano", "dir", "kind", "ssrc", "pt", "seq", "rtp_ts", "size",
		"recovered", "too_late",
	})
	_ = l.summaryW.Write([]string{
		"metric", "value",
	})
	l.packetsW.Flush()
	l.summaryW.Flush()

	return l, nil
}

func (l *CSVLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.packetsW.Flush()
	l.summaryW.Flush()
	_ = l.packetsF.Close()
	_ = l.summaryF.Close()
}

func (l *CSVLogger) Packet(dir, kind string, now time.Time, ssrc uint32, pt uint8, seq uint16, rtpTs uint32, size int, recovered, tooLate bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_ = l.packetsW.Write([]string{
		strconv.FormatInt(now.UnixNano(), 10),
		dir,
		kind,
		strconv.FormatUint(uint64(ssrc), 10),
		strconv.FormatUint(uint64(pt), 10),
		strconv.FormatUint(uint64(seq), 10),
		strconv.FormatUint(uint64(rtpTs), 10),
		strconv.Itoa(size),
		strconv.FormatBool(recovered),
		strconv.FormatBool(tooLate),
	})
	l.packetsW.Flush()
}

func (l *CSVLogger) Summary(metric string, value any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.summaryW.Write([]string{metric, toString(value)})
	l.summaryW.Flush()
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', 6, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return ""
	}
}
