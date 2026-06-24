package process

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Command wraps a cancellable child process started by dev mode.
type Command struct {
	cancel context.CancelFunc
	done   chan error
}

// Start launches a configured local service command and streams stdout/stderr
// into shared status logs.
func Start(ctx context.Context, dir, command string, status *Status) (*Command, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, nil
	}
	var setState func(string)
	if status != nil {
		setState = status.SetServer
	}
	return StartArgs(ctx, dir, parts[0], parts[1:], "backend", status, setState)
}

// StartArgs launches a child process from explicit argv parts. The optional
// setState hook lets callers connect the process lifecycle to status fields.
func StartArgs(ctx context.Context, dir, name string, args []string, label string, status *Status, setState func(string)) (*Command, error) {
	runCtx, cancel := context.WithCancel(ctx)
	if name == "" {
		cancel()
		return nil, nil
	}
	done := make(chan error, 1)
	cmd := exec.CommandContext(runCtx, name, args...)
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
	if setState != nil {
		setState("running")
	}
	if status != nil {
		status.AppendLog(label + " started: " + strings.Join(append([]string{name}, args...), " "))
	}
	go scanLines(stdout, status)
	go scanLines(stderr, status)
	go func() {
		err := cmd.Wait()
		if runCtx.Err() != nil {
			if setState != nil {
				setState("stopped")
			}
			done <- nil
			return
		}
		if err != nil {
			if setState != nil {
				setState("failed")
			}
			if status != nil {
				status.AppendLog(label + " failed: " + err.Error())
			}
			done <- fmt.Errorf("%s failed: %w", label, err)
			return
		}
		if setState != nil {
			setState("stopped")
		}
		done <- nil
	}()
	return &Command{cancel: cancel, done: done}, nil
}

// Stop cancels the command context. exec.CommandContext handles terminating the
// child process.
func (c *Command) Stop() {
	if c != nil && c.cancel != nil {
		c.cancel()
	}
}

// Done reports child process completion. A nil error usually means the command
// stopped because its context was canceled.
func (c *Command) Done() <-chan error {
	if c == nil {
		done := make(chan error)
		close(done)
		return done
	}
	return c.done
}

// scanLines keeps child process output visible in the dashboard without
// coupling the runner to any terminal UI.
func scanLines(r interface{ Read([]byte) (int, error) }, status *Status) {
	if status == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		status.AppendLog(scanner.Text())
	}
}
