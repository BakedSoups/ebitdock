package process

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"ebitdock/internal/config"
)

type Status struct {
	mu sync.RWMutex

	logFile string
	state
}

type state struct {
	Project       string    `json:"project"`
	WebPort       int       `json:"webPort"`
	DashboardPort int       `json:"dashboardPort"`
	ServerEnabled bool      `json:"serverEnabled"`
	ServerPort    int       `json:"serverPort"`
	ServerStatus  string    `json:"serverStatus"`
	BuildStatus   string    `json:"buildStatus"`
	BuildDuration string    `json:"buildDuration"`
	LastBuildAt   time.Time `json:"lastBuildAt"`
	CurrentErrors []string  `json:"currentErrors"`
	Watched       []string  `json:"watched"`
	Logs          []string  `json:"logs"`
}

func (s *Status) SetLogFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logFile = path
}

func NewStatus(cfg config.Config) *Status {
	return &Status{
		state: state{
			Project:       cfg.Project,
			WebPort:       cfg.WebPort(),
			DashboardPort: cfg.DashboardPort(),
			ServerEnabled: cfg.APIEnabled(),
			ServerPort:    cfg.APIPort(),
			ServerStatus:  "disabled",
			BuildStatus:   "idle",
			Watched:       cfg.WatchPatterns(),
		},
	}
}

func (s *Status) Snapshot() state {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := s.state
	cp.CurrentErrors = append([]string(nil), s.CurrentErrors...)
	cp.Watched = append([]string(nil), s.Watched...)
	cp.Logs = append([]string(nil), s.Logs...)
	return cp
}

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

func (s *Status) SetServer(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ServerStatus = status
}

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

func (s *Status) RecentLogs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.Logs...)
}
