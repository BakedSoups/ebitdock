package cliui

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"ebitdock/internal/config"
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
