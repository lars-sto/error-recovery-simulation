package sim

// MultiRecorder fan-outs samples to multiple Recorder implementations
type multiRecorder struct {
	rs []Recorder
}

// MultiRecorder creates a Recorder that forwards OnSample/Close to all recorders
func MultiRecorder(rs ...Recorder) Recorder {
	out := &multiRecorder{rs: make([]Recorder, 0, len(rs))}
	for _, r := range rs {
		if r != nil {
			out.rs = append(out.rs, r)
		}
	}
	return out
}

func (m *multiRecorder) OnSample(s TimeSample) {
	for _, r := range m.rs {
		r.OnSample(s)
	}
}

func (m *multiRecorder) Close() error {
	var firstErr error
	for _, r := range m.rs {
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
