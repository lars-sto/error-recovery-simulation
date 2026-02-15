module github.com/lars-sto/error-recovery-simulation

go 1.25

require (
    github.com/pion/interceptor v0.1.44
    github.com/lars-sto/adaptive-error-recovery-controller v0.0.0
)

replace github.com/pion/interceptor => ../pion/forks/interceptor
replace github.com/lars-sto/adaptive-error-recovery-controller => ../adaptive-error-recovery-controller