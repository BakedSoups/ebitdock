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

// projectData is the small view model used by ebitdock.yaml.tmpl.
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

// InitProject creates an ebitdock.yaml either in the current directory or in a
// new named directory. It intentionally does not scaffold the user's web app.
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
		"ebitdock.yaml": mustRender(configTemplate, data),
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
	fmt.Println("  edit ebitdock.yaml")
	return nil
}

// InitCurrentProject writes ebitdock.yaml into the current repo, preserving an
// existing file so init can be rerun safely.
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
	fmt.Println("  edit ebitdock.yaml")
	return nil
}

// shouldInitCurrent treats "ebitdock init <current-folder>" inside a Go module
// as an in-place init instead of trying to create a nested folder.
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

// inferRebuildWatchPatterns chooses common Go game source/asset directories for
// a first config. Users are expected to edit the YAML after init.
func inferRebuildWatchPatterns(root string) []string {
	candidates := []struct {
		dir     string
		pattern string
	}{
		{"cmd", "./cmd/**/*.go"},
		{"wasm", "./wasm/**/*.go"},
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

// inferStaticWatchPatterns watches the detected browser static root.
func inferStaticWatchPatterns(root string) []string {
	return []string{inferStaticRoot(root) + "/**"}
}

// inferStaticRoot prefers existing conventional static roots and falls back to
// ./static for new/basic projects.
func inferStaticRoot(root string) string {
	for _, dir := range []string{"static", "web", "public"} {
		if info, err := os.Stat(filepath.Join(root, dir)); err == nil && info.IsDir() {
			return "./" + dir
		}
	}
	return "./static"
}

// inferGamePackage guesses the most likely game entrypoint package.
func inferGamePackage(root, name string) string {
	candidates := []string{
		filepath.Join("cmd", name),
		filepath.Join("cmd", "game"),
		"wasm",
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

// hasGoFiles reports whether a directory looks like a Go package.
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

// writeNewFile creates a file without overwriting user-owned content.
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

// validProjectName keeps generated folder names portable and predictable.
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

// mustRender renders embedded templates during init. Panics here indicate a
// developer error in a checked-in template, not user input.
func mustRender(src string, data projectData) string {
	tpl := template.Must(template.New("file").Parse(src))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}
