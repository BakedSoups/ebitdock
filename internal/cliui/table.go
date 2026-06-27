package cliui

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/BakedSoups/ebitdock/internal/config"
)

// DevStatus prints the startup service table. tabwriter keeps the output
// Docker-like without adding a formatting dependency.
func DevStatus(w io.Writer, cfg config.Config, webService string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVICE\tSTATUS\tURL/DETAILS")
	fmt.Fprintf(tw, "%s\trunning\t%s\n", webService, urls(cfg.WebPorts()))
	for _, name := range sortedServiceNames(cfg.EnabledServices()) {
		if name == "web" {
			continue
		}
		service := cfg.EnabledServices()[name]
		fmt.Fprintf(tw, "%s\trunning\t%s\n", name, ports(service.Ports))
	}
	fmt.Fprintf(tw, "dashboard\trunning\thttp://localhost:%d\n", cfg.DashboardPort())
	fmt.Fprintf(tw, "watch\tactive\t%d patterns\n", len(cfg.WatchPatterns()))
	_ = tw.Flush()
}

// Progress prints a compact Docker-like activity row for blocking steps such
// as tests and WASM builds.
func Progress(w io.Writer, name, status, detail string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\t%s %s\n", name, status, progressBar(status), detail)
	_ = tw.Flush()
}

// Activity keeps a single terminal row moving while a blocking step runs.
type Activity struct {
	w      io.Writer
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
	name   string
	status string
	detail string
	start  time.Time
}

func StartActivity(w io.Writer, name, status, detail string) *Activity {
	a := &Activity{
		w:      w,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
		name:   name,
		status: status,
		detail: detail,
		start:  time.Now(),
	}
	a.print(0)
	go a.run()
	return a
}

func (a *Activity) Stop() {
	a.once.Do(func() {
		close(a.stop)
		<-a.done
		fmt.Fprint(a.w, "\r\033[K")
	})
}

func (a *Activity) run() {
	defer close(a.done)
	ticker := time.NewTicker(180 * time.Millisecond)
	defer ticker.Stop()
	frame := 1
	for {
		select {
		case <-a.stop:
			return
		case <-ticker.C:
			a.print(frame)
			frame++
		}
	}
}

func (a *Activity) print(frame int) {
	elapsed := time.Since(a.start).Truncate(100 * time.Millisecond)
	fmt.Fprintf(a.w, "\r\033[K%-8s  %-8s  %s %s  %s", a.name, a.status, activityBar(frame), elapsed, a.detail)
}

// Result prints the final row for a blocking step.
func Result(w io.Writer, name string, duration time.Duration, err error) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err != nil {
		fmt.Fprintf(tw, "%s\tfailed\t%s %s\n", name, progressBar("failed"), err)
	} else {
		fmt.Fprintf(tw, "%s\tok\t%s %s\n", name, progressBar("ok"), duration.String())
	}
	_ = tw.Flush()
}

func sortedServiceNames(services map[string]config.ServiceConfig) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func urls(portList []config.PortConfig) string {
	if len(portList) == 0 {
		return "-"
	}
	items := make([]string, 0, len(portList))
	for _, port := range portList {
		items = append(items, port.URL)
	}
	return strings.Join(items, ", ")
}

func ports(portList []config.PortConfig) string {
	if len(portList) == 0 {
		return "-"
	}
	items := make([]string, 0, len(portList))
	for _, port := range portList {
		items = append(items, fmt.Sprintf("%s :%d", port.Name, port.Port))
	}
	return strings.Join(items, ", ")
}

func progressBar(status string) string {
	switch status {
	case "ok":
		return "[==========]"
	case "failed":
		return "[xxx-------]"
	default:
		return "[...-------]"
	}
}

func activityBar(frame int) string {
	const width = 10
	const pulse = 3
	pos := frame % (width + pulse)
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < width; i++ {
		if i >= pos-pulse && i < pos {
			b.WriteByte('=')
		} else {
			b.WriteByte('-')
		}
	}
	b.WriteByte(']')
	return b.String()
}
