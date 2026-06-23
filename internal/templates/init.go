package templates

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed files/ebitdock.yaml.tmpl
var configTemplate string

//go:embed files/game-main.go.tmpl
var gameMainTemplate string

//go:embed files/game-bridge-js.go
var gameBridgeJS string

//go:embed files/game-bridge-stub.go
var gameBridgeStub string

//go:embed files/game-go.mod.tmpl
var gameModTemplate string

//go:embed files/index.html.tmpl
var indexTemplate string

//go:embed files/bridge.js
var bridgeJS string

//go:embed files/style.css
var styleCSS string

//go:embed files/server-main.go.tmpl
var serverMainTemplate string

//go:embed files/server-go.mod.tmpl
var serverModTemplate string

type projectData struct {
	Name           string
	Module         string
	GamePackage    string
	GameOutput     string
	WASMExec       string
	StaticRoot     string
	WebPort        int
	DashboardPort  int
	WatchPatterns  []string
	StaticPatterns []string
}

func InitProject(name string) error {
	if name == "" || name == "." {
		return InitCurrentProject(name)
	}
	if !validProjectName(name) {
		return fmt.Errorf("invalid project name %q: use letters, numbers, dash, or underscore", name)
	}
	if shouldInitCurrent(name) {
		return InitCurrentProject(name)
	}
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("%s already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	data := projectData{
		Name:           name,
		Module:         "example.com/" + strings.ReplaceAll(name, "-", ""),
		GamePackage:    "./game",
		GameOutput:     "./static/game.wasm",
		WASMExec:       "./static/wasm_exec.js",
		StaticRoot:     "./static",
		WebPort:        8080,
		DashboardPort:  8081,
		WatchPatterns:  []string{"./game/**/*.go", "./assets/**"},
		StaticPatterns: []string{"./static/**"},
	}
	files := map[string]string{
		"ebitdock.yaml":       mustRender(configTemplate, data),
		"game/main.go":        mustRender(gameMainTemplate, data),
		"game/bridge_js.go":   gameBridgeJS,
		"game/bridge_stub.go": gameBridgeStub,
		"game/go.mod":         mustRender(gameModTemplate, data),
		"static/index.html":   mustRender(indexTemplate, data),
		"static/bridge.js":    bridgeJS,
		"static/style.css":    styleCSS,
		"server/main.go":      mustRender(serverMainTemplate, data),
		"server/go.mod":       mustRender(serverModTemplate, data),
		"assets/.gitkeep":     "",
		"static/.gitkeep":     "",
	}

	for path, content := range files {
		full := filepath.Join(name, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("created %s\n", name)
	fmt.Println("next:")
	fmt.Printf("  cd %s\n", name)
	fmt.Println("  ebitdock dev")
	return nil
}

func InitCurrentProject(name string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if name == "" || name == "." {
		name = filepath.Base(root)
	}
	if !validProjectName(name) {
		return fmt.Errorf("invalid project name %q: use letters, numbers, dash, or underscore", name)
	}

	data := projectData{
		Name:           name,
		Module:         "example.com/" + strings.ReplaceAll(name, "-", ""),
		GamePackage:    inferGamePackage(root, name),
		StaticRoot:     inferStaticRoot(root),
		WebPort:        8080,
		DashboardPort:  8081,
		WatchPatterns:  inferRebuildWatchPatterns(root),
		StaticPatterns: inferStaticWatchPatterns(root),
	}
	data.GameOutput = "./" + filepath.ToSlash(filepath.Join(strings.TrimPrefix(data.StaticRoot, "./"), "game.wasm"))
	data.WASMExec = "./" + filepath.ToSlash(filepath.Join(strings.TrimPrefix(data.StaticRoot, "./"), "wasm_exec.js"))
	files := map[string]string{
		"ebitdock.yaml": mustRender(configTemplate, data),
	}
	written := 0
	skipped := 0
	for path, content := range files {
		ok, err := writeNewFile(filepath.Join(root, path), content)
		if err != nil {
			return err
		}
		if ok {
			written++
		} else {
			skipped++
		}
	}
	fmt.Printf("initialized %s in %s (%d written, %d kept)\n", name, root, written, skipped)
	fmt.Println("next:")
	fmt.Println("  ebitdock dev")
	return nil
}

func shouldInitCurrent(name string) bool {
	root, err := os.Getwd()
	if err != nil {
		return false
	}
	if filepath.Base(root) != name {
		return false
	}
	_, err = os.Stat(filepath.Join(root, "go.mod"))
	return err == nil
}

func inferRebuildWatchPatterns(root string) []string {
	candidates := []struct {
		dir     string
		pattern string
	}{
		{"cmd", "./cmd/**/*.go"},
		{"internal", "./internal/**/*.go"},
		{"assets", "./assets/**"},
		{"levels", "./levels/**"},
		{"soundeffects", "./soundeffects/**"},
	}
	var patterns []string
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(root, candidate.dir)); err == nil && info.IsDir() {
			patterns = append(patterns, candidate.pattern)
		}
	}
	if len(patterns) == 0 {
		return []string{"./**/*.go", "./assets/**"}
	}
	return patterns
}

func inferStaticWatchPatterns(root string) []string {
	return []string{inferStaticRoot(root) + "/**"}
}

func inferStaticRoot(root string) string {
	for _, dir := range []string{"static", "web", "public"} {
		if info, err := os.Stat(filepath.Join(root, dir)); err == nil && info.IsDir() {
			return "./" + dir
		}
	}
	return "./static"
}

func inferGamePackage(root, name string) string {
	candidates := []string{
		filepath.Join("cmd", name),
		filepath.Join("cmd", "game"),
		"cmd",
		".",
	}
	for _, candidate := range candidates {
		if hasGoFiles(filepath.Join(root, candidate)) {
			return "./" + filepath.ToSlash(candidate)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(root, "cmd", "*", "main.go"))
	if len(matches) > 0 {
		dir, err := filepath.Rel(root, filepath.Dir(matches[0]))
		if err == nil {
			return "./" + filepath.ToSlash(dir)
		}
	}
	return "."
}

func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

func writeNewFile(path, content string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(content), 0o644)
}

func validProjectName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func mustRender(src string, data projectData) string {
	tpl := template.Must(template.New("file").Parse(src))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}
