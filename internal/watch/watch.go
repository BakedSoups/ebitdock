package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func Changes(ctx context.Context, root string, patterns []string) (<-chan string, <-chan error, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}
	changes := make(chan string, 16)
	errs := make(chan error, 1)

	for _, dir := range watchDirs(root, patterns) {
		if err := addRecursive(watcher, dir); err != nil {
			_ = watcher.Close()
			return nil, nil, err
		}
	}

	go func() {
		defer close(changes)
		defer close(errs)
		defer watcher.Close()
		var last string
		var lastAt time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-watcher.Errors:
				if err != nil {
					errs <- err
				}
			case event := <-watcher.Events:
				if event.Name == "" {
					continue
				}
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = addRecursive(watcher, event.Name)
					}
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}
				if event.Name == last && time.Since(lastAt) < 250*time.Millisecond {
					continue
				}
				last, lastAt = event.Name, time.Now()
				changes <- event.Name
			}
		}
	}()
	return changes, errs, nil
}

func watchDirs(root string, patterns []string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, pattern := range patterns {
		p := strings.TrimPrefix(pattern, "./")
		base := p
		if i := strings.Index(base, "**"); i >= 0 {
			base = base[:i]
		}
		if i := strings.IndexAny(base, "*?["); i >= 0 {
			base = base[:i]
		}
		base = strings.TrimSuffix(base, "/")
		if base == "" {
			base = "."
		}
		dir := filepath.Join(root, base)
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func addRecursive(watcher *fsnotify.Watcher, dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == ".git" || name == "dist" {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}
