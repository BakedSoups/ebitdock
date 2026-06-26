package doctor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"ebitdock/internal/config"
	"ebitdock/internal/docker"
	"ebitdock/internal/tools"
)

// Run checks the local project and toolchain needed by ebitdock commands.
func Run(w io.Writer, root string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CHECK\tSTATUS\tDETAILS")

	cfg, err := config.Load(filepath.Join(root, "ebitdock.yaml"))
	if err != nil {
		fmt.Fprintf(tw, "config\tfailed\t%v\n", err)
		_ = tw.Flush()
		return errors.New("doctor found problems")
	}
	fmt.Fprintln(tw, "config\tok\tebitdock.yaml")

	hasProblems := false
	if detail, err := goVersion(); err != nil {
		fmt.Fprintf(tw, "go\tfailed\t%v\n", err)
		hasProblems = true
	} else {
		fmt.Fprintf(tw, "go\tok\t%s\n", detail)
	}

	if cfg.DockerEnabled() {
		if path, err := docker.CheckDocker(nil); err != nil {
			fmt.Fprintf(tw, "docker\tfailed\t%v\n", err)
			hasProblems = true
		} else {
			fmt.Fprintf(tw, "docker\tok\t%s\n", path)
		}
		fmt.Fprintf(tw, "compose\tok\t%s\n", cfg.ComposeFile())
	} else if cfg.UsesWebCommand() {
		fmt.Fprintf(tw, "wasmserve\tskipped\tservices.web.command: %s\n", cfg.WebCommand())
	} else {
		if path, err := exec.LookPath("wasmserve"); err != nil {
			fmt.Fprintf(tw, "wasmserve\tfailed\tinstall with: %s\n", tools.WasmserveInstallCommand)
			hasProblems = true
		} else {
			fmt.Fprintf(tw, "wasmserve\tok\t%s\n", path)
		}
	}

	if detail, ok := packageDetail(root, cfg.Game.Package); !ok {
		fmt.Fprintf(tw, "game\tfailed\t%s\n", detail)
		hasProblems = true
	} else {
		fmt.Fprintf(tw, "game\tok\t%s\n", detail)
	}

	staticPath := filepath.Join(root, cfg.StaticRoot())
	if existsDir(staticPath) {
		fmt.Fprintf(tw, "web\tok\t%s %s\n", cfg.StaticRoot(), portDetails(cfg.WebPorts()))
	} else {
		fmt.Fprintf(tw, "web\twarn\t%s does not exist at %s\n", cfg.StaticRoot(), staticPath)
	}
	if cfg.UsesWebCommand() {
		fmt.Fprintln(tw, "shell\tskipped\tproject web command owns browser dev flow")
	} else if hints := tools.BrowserShellHints(root, cfg.StaticRoot()); len(hints) == 0 {
		fmt.Fprintln(tw, "shell\tok\twasmserve dev hooks")
	} else {
		for _, hint := range hints {
			fmt.Fprintf(tw, "shell\twarn\t%s\n", hint)
		}
	}

	fmt.Fprintf(tw, "dashboard\tok\t:%d\n", cfg.DashboardPort())
	if cfg.APIEnabled() {
		if strings.TrimSpace(cfg.APICommand()) == "" {
			fmt.Fprintln(tw, "api\tfailed\tenabled but command is empty")
			hasProblems = true
		} else {
			fmt.Fprintf(tw, "api\tok\t%s %s\n", portDetails(cfg.APIPorts()), cfg.APICommand())
		}
	} else {
		fmt.Fprintln(tw, "api\tdisabled\t-")
	}

	_ = tw.Flush()
	if hasProblems {
		return errors.New("doctor found problems")
	}
	return nil
}

func goVersion() (string, error) {
	path, err := exec.LookPath("go")
	if err != nil {
		return "", errors.New("go executable not found in PATH")
	}
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func packageDetail(root, pkg string) (string, bool) {
	if pkg == "" {
		return "game.package is empty", false
	}
	if strings.HasPrefix(pkg, ".") || strings.HasPrefix(pkg, string(filepath.Separator)) {
		dir := filepath.Join(root, pkg)
		if existsDir(dir) {
			return pkg, true
		}
		return pkg + " does not exist", false
	}
	return pkg, true
}

func existsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func portDetails(ports []config.PortConfig) string {
	if len(ports) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(ports))
	for _, port := range ports {
		parts = append(parts, fmt.Sprintf("%s :%d", port.Name, port.Port))
	}
	return strings.Join(parts, ",")
}
