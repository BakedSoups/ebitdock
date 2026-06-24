package process

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
)

// Command wraps a cancellable child process started by dev mode.
type Command struct {
	cancel context.CancelFunc
}

// Start launches a configured local service command and streams stdout/stderr
// into shared status logs.
func Start(ctx context.Context, dir, command string, status *Status) (*Command, error) {
	runCtx, cancel := context.WithCancel(ctx)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		cancel()
		return nil, nil
	}
	cmd := exec.CommandContext(runCtx, parts[0], parts[1:]...)
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	status.SetServer("running")
	status.AppendLog("backend started: " + command)
	go scanLines(stdout, status)
	go scanLines(stderr, status)
	go func() {
		err := cmd.Wait()
		if runCtx.Err() != nil {
			status.SetServer("stopped")
			return
		}
		if err != nil {
			status.SetServer("failed")
			status.AppendLog("backend failed: " + err.Error())
			return
		}
		status.SetServer("stopped")
	}()
	return &Command{cancel: cancel}, nil
}

// Stop cancels the command context. exec.CommandContext handles terminating the
// child process.
func (c *Command) Stop() {
	if c != nil && c.cancel != nil {
		c.cancel()
	}
}

// scanLines keeps child process output visible in the dashboard without
// coupling the runner to any terminal UI.
func scanLines(r interface{ Read([]byte) (int, error) }, status *Status) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		status.AppendLog(scanner.Text())
	}
}
