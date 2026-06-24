package process

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"ebitdock/internal/config"
)

type PortStatus = config.PortConfig

// Status is the shared mutable state behind terminal logs and dashboard JSON.
// All reads and writes go through methods so dev goroutines can update it
// safely.
type Status struct {
	mu sync.RWMutex

	logFile string
	state
}

// state is the mutex-free snapshot shape encoded by /api/status.
type state struct {
	Project       string       `json:"project"`
	WebPort       int          `json:"webPort"`
	WebPorts      []PortStatus `json:"webPorts"`
	DashboardPort int          `json:"dashboardPort"`
	ServerEnabled bool         `json:"serverEnabled"`
	ServerPort    int          `json:"serverPort"`
	ServerPorts   []PortStatus `json:"serverPorts"`
	ServerStatus  string       `json:"serverStatus"`
	BuildStatus   string       `json:"buildStatus"`
	BuildDuration string       `json:"buildDuration"`
	LastBuildAt   time.Time    `json:"lastBuildAt"`
	CurrentErrors []string     `json:"currentErrors"`
	Watched       []string     `json:"watched"`
	Logs          []string     `json:"logs"`
}

// SetLogFile enables persistent project-local logging.
func (s *Status) SetLogFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logFile = path
}

// NewStatus seeds dashboard state from normalized project config.
func NewStatus(cfg config.Config) *Status {
	return &Status{
		state: state{
			Project:       cfg.Project,
			WebPort:       cfg.WebPort(),
			WebPorts:      cfg.WebPorts(),
			DashboardPort: cfg.DashboardPort(),
			ServerEnabled: cfg.APIEnabled(),
			ServerPort:    cfg.APIPort(),
			ServerPorts:   cfg.APIPorts(),
			ServerStatus:  "disabled",
			BuildStatus:   "idle",
			Watched:       cfg.WatchPatterns(),
		},
	}
}

// Snapshot returns a copy that is safe to encode or inspect without holding the
// status lock.
func (s *Status) Snapshot() state {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := s.state
	cp.CurrentErrors = append([]string(nil), s.CurrentErrors...)
	cp.WebPorts = append([]PortStatus(nil), s.WebPorts...)
	cp.ServerPorts = append([]PortStatus(nil), s.ServerPorts...)
	cp.Watched = append([]string(nil), s.Watched...)
	cp.Logs = append([]string(nil), s.Logs...)
	return cp
}

// SetBuild records the latest WASM build state and any current error.
func (s *Status) SetBuild(status, duration string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BuildStatus = status
	s.BuildDuration = duration
	s.LastBuildAt = time.Now()
	if err != nil {
		s.CurrentErrors = []string{err.Error()}
	} else {
		s.CurrentErrors = nil
	}
}

// SetServer records the optional backend process state.
func (s *Status) SetServer(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ServerStatus = status
}

// AppendLog appends to the in-memory ring and mirrors the same line to disk
// when persistent logging is enabled.
func (s *Status) AppendLog(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := time.Now().Format("15:04:05") + " " + line
	s.Logs = append(s.Logs, entry)
	if len(s.Logs) > 200 {
		s.Logs = s.Logs[len(s.Logs)-200:]
	}
	if s.logFile != "" {
		_ = os.MkdirAll(filepath.Dir(s.logFile), 0o755)
		f, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			_, _ = f.WriteString(entry + "\n")
			_ = f.Close()
		}
	}
}

// RecentLogs returns a copy of the current in-memory log tail.
func (s *Status) RecentLogs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.Logs...)
}
