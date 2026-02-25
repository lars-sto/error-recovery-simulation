package sim

// SummaryRecorder aggregates per-sample stats into thesis-friendly summary metrics
// It implements Recorder so it can be plugged into RunScenario without modifying the runner
type SummaryRecorder struct {
	// sample aggregates
	samples         int64
	sumQueueDelayMs float64

	enabledSamples int64
	sumPolicyR     float64
	sumPolicyOh    float64
	maxPolicyR     uint32

	// track loss window seen by recorder (debug/analysis)
	sumLossWindow float64
	maxLossWindow float64
}

func NewSummaryRecorder() *SummaryRecorder { return &SummaryRecorder{} }

func (r *SummaryRecorder) OnSample(s TimeSample) {
	r.samples++
	r.sumQueueDelayMs += s.QueueDelayMs

	r.sumLossWindow += s.LossWindow
	if s.LossWindow > r.maxLossWindow {
		r.maxLossWindow = s.LossWindow
	}

	if s.PolicyEnabled {
		r.enabledSamples++
		r.sumPolicyR += float64(s.PolicyR)
		r.sumPolicyOh += s.PolicyOverhead
		if s.PolicyR > r.maxPolicyR {
			r.maxPolicyR = s.PolicyR
		}
	}
}

func (r *SummaryRecorder) Close() error { return nil }

// MeanQueueDelayMs is averaged across all samples
func (r *SummaryRecorder) MeanQueueDelayMs() float64 {
	if r.samples <= 0 {
		return 0
	}
	return r.sumQueueDelayMs / float64(r.samples)
}

// MeanPolicyR is averaged across samples where PolicyEnabled==true
func (r *SummaryRecorder) MeanPolicyR() float64 {
	if r.enabledSamples <= 0 {
		return 0
	}
	return r.sumPolicyR / float64(r.enabledSamples)
}

func (r *SummaryRecorder) MaxPolicyR() uint32 { return r.maxPolicyR }

// MeanPolicyOverhead is averaged across samples where PolicyEnabled==true
func (r *SummaryRecorder) MeanPolicyOverhead() float64 {
	if r.enabledSamples <= 0 {
		return 0
	}
	return r.sumPolicyOh / float64(r.enabledSamples)
}

// MeanLossWindow is averaged across all samples (pre-FEC window loss ratio)
func (r *SummaryRecorder) MeanLossWindow() float64 {
	if r.samples <= 0 {
		return 0
	}
	return r.sumLossWindow / float64(r.samples)
}

func (r *SummaryRecorder) MaxLossWindow() float64 { return r.maxLossWindow }
