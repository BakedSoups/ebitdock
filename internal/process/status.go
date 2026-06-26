package process

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"ebitdock/internal/config"
)

type PortStatus = config.PortConfig

type ServiceStatus struct {
	Name   string       `json:"name"`
	Status string       `json:"status"`
	Ports  []PortStatus `json:"ports"`
}

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
	Project       string          `json:"project"`
	WebPort       int             `json:"webPort"`
	WebPorts      []PortStatus    `json:"webPorts"`
	DashboardPort int             `json:"dashboardPort"`
	ServerEnabled bool            `json:"serverEnabled"`
	ServerPort    int             `json:"serverPort"`
	ServerPorts   []PortStatus    `json:"serverPorts"`
	ServerStatus  string          `json:"serverStatus"`
	Services      []ServiceStatus `json:"services"`
	CheckEnabled  bool            `json:"checkEnabled"`
	CheckCommand  string          `json:"checkCommand"`
	CheckStatus   string          `json:"checkStatus"`
	CheckDuration string          `json:"checkDuration"`
	BuildStatus   string          `json:"buildStatus"`
	BuildDuration string          `json:"buildDuration"`
	BuildLog      string          `json:"buildLog"`
	LastChange    string          `json:"lastChange"`
	LastBuildAt   time.Time       `json:"lastBuildAt"`
	CurrentErrors []string        `json:"currentErrors"`
	Watched       []string        `json:"watched"`
	Logs          []string        `json:"logs"`
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
			Services:      serviceStatuses(cfg),
			CheckEnabled:  cfg.BeforeRebuildCheckEnabled(),
			CheckCommand:  cfg.BeforeRebuildCheckCommand(),
			CheckStatus:   "idle",
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
	cp.Services = copyServices(s.Services)
	cp.Watched = append([]string(nil), s.Watched...)
	cp.Logs = append([]string(nil), s.Logs...)
	return cp
}

// SetCheck records the latest configured command check state.
func (s *Status) SetCheck(status, duration string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CheckStatus = status
	s.CheckDuration = duration
	if err != nil {
		s.CurrentErrors = []string{err.Error()}
	} else {
		s.CurrentErrors = nil
	}
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

// SetBuildLog records the latest raw build output for dashboard debugging.
func (s *Status) SetBuildLog(logText string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BuildLog = logText
}

// SetLastChange records the latest watched path that triggered dev activity.
func (s *Status) SetLastChange(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastChange = path
}

// SetServer records the optional backend process state.
func (s *Status) SetServer(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ServerStatus = status
	for i := range s.Services {
		if s.Services[i].Name == "api" {
			s.Services[i].Status = status
		}
	}
}

// SetServices records the same status for all configured Docker services.
func (s *Status) SetServices(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Services {
		s.Services[i].Status = status
	}
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

func serviceStatuses(cfg config.Config) []ServiceStatus {
	services := cfg.EnabledServices()
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ServiceStatus, 0, len(names))
	for _, name := range names {
		service := services[name]
		out = append(out, ServiceStatus{
			Name:   name,
			Status: "configured",
			Ports:  append([]PortStatus(nil), service.Ports...),
		})
	}
	return out
}

func copyServices(in []ServiceStatus) []ServiceStatus {
	out := append([]ServiceStatus(nil), in...)
	for i := range out {
		out[i].Ports = append([]PortStatus(nil), out[i].Ports...)
	}
	return out
}
