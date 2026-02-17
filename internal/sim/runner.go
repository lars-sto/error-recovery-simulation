package sim

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Runner struct {
	cfg RunnerConfig
}

func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{cfg: cfg}
}

func (r *Runner) RunScenario(sc Scenario) error {
	// Pro Scenario: zwei Runs
	for _, mode := range []Mode{ModeStatic, ModeAdaptive} {
		runID := fmt.Sprintf("%s__%s__%d", sc.Name, mode, sc.Seed)
		out := filepath.Join(r.cfg.OutDir, runID)
		if err := os.MkdirAll(out, 0o755); err != nil {
			return err
		}
		if err := r.runOnce(sc, mode, out); err != nil {
			return fmt.Errorf("runOnce mode=%s: %w", mode, err)
		}
	}
	return nil
}

func (r *Runner) runOnce(sc Scenario, mode Mode, outDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), sc.Duration+2*time.Second)
	defer cancel()

	logger, err := NewCSVLogger(outDir)
	if err != nil {
		return err
	}
	defer logger.Close()

	env, err := NewEnv(sc, mode, logger)
	if err != nil {
		return err
	}
	defer env.Close()

	// Start receiver first
	doneRecv := make(chan error, 1)
	go func() { doneRecv <- env.RunReceiver(ctx) }()

	// Start sender
	doneSend := make(chan error, 1)
	go func() { doneSend <- env.RunSender(ctx) }()

	// Wait (either ends first -> cancel -> wait both)
	var firstErr error
	select {
	case err := <-doneSend:
		firstErr = err
	case err := <-doneRecv:
		firstErr = err
	case <-ctx.Done():
		firstErr = ctx.Err()
	}

	cancel()
	errS := <-doneSend
	errR := <-doneRecv

	// summary
	env.EmitSummary()

	if firstErr != nil && firstErr != context.Canceled && firstErr != context.DeadlineExceeded {
		return firstErr
	}
	if errS != nil && errS != context.Canceled && errS != context.DeadlineExceeded {
		return errS
	}
	if errR != nil && errR != context.Canceled && errR != context.DeadlineExceeded {
		return errR
	}
	return nil
}
