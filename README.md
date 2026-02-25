# Error Recovery Simulation Framework

A simulation and integration environment for evaluating adaptive FEC policies in WebRTC systems.

This repository integrates:
- A forked Pion flexfec interceptor
- The adaptive-error-recovery-controller
- A runtime adapter layer
- Network simulation / experimentation tools


## Architecture

```
Network Simulation
        ↓
StatsSource
        ↓
Adaptive Controller (AERC)
        ↓
PolicyDecision
        ↓
Adapter
        ↓
RuntimeBus (ConfigSource)
        ↓
Pion FlexFEC Interceptor
        ↓
RTP Output
```

## Usage

```batch
go run ./cmd/simulate/batch \
  -runs 50 \
  -seed 1 \
  -out results/summary.csv \
  -csvdir results/timeseries \
  -timeseries bwe_bottleneck
```
Creates a summary of all runs and detailed time-series 

### Python
```
python3 -m venv .venv
source .venv/bin/activate

```