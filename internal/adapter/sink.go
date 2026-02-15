package adapter

import "github.com/lars-sto/adaptive-error-recovery-controller/recovery"

type SinkFunc func(recovery.PolicyDecision)

func (f SinkFunc) Publish(d recovery.PolicyDecision) { f(d) }
