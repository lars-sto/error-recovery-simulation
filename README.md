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

