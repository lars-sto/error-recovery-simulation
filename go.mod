module github.com/lars-sto/error-recovery-simulation

go 1.25

require (
	github.com/lars-sto/adaptive-error-recovery-controller v0.0.0
	github.com/pion/interceptor v0.1.44
	github.com/pion/rtp v1.8.26
	github.com/pion/transport/v4 v4.0.1
)

require (
	github.com/pion/logging v0.2.4 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.16 // indirect
	golang.org/x/time v0.10.0 // indirect
)

replace github.com/pion/interceptor => ../pion/forks/interceptor

replace github.com/lars-sto/adaptive-error-recovery-controller => ../adaptive-error-recovery-controller
