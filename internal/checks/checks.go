package checks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/BakedSoups/ebitdock/internal/process"
)

// Result captures the outcome of an optional pre-rebuild command.
type Result struct {
	Duration time.Duration
	Log      string
	Err      error
}

// Run executes a configured check command from the project root and mirrors the
// result into dashboard status. The command is split like other ebitdock
// commands: simple shell-style words, no shell expansion.
func Run(ctx context.Context, root, command string, status *process.Status) Result {
	start := time.Now()
	status.SetCheck("running", "", nil)
	status.AppendLog("running check: " + command)

	parts := strings.Fields(command)
	if len(parts) == 0 {
		err := fmt.Errorf("check command is empty")
		return finish(status, start, "", err)
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return finish(status, start, out.String(), fmt.Errorf("check failed: %w", err))
	}
	return finish(status, start, out.String(), nil)
}

func finish(status *process.Status, start time.Time, logText string, err error) Result {
	duration := time.Since(start)
	if logText != "" {
		status.AppendLog(logText)
	}
	if err != nil {
		status.SetCheck("failed", duration.String(), err)
		status.AppendLog(err.Error())
		return Result{Duration: duration, Log: logText, Err: err}
	}
	status.SetCheck("ok", duration.String(), nil)
	status.AppendLog("check completed in " + duration.String())
	return Result{Duration: duration, Log: logText}
}
