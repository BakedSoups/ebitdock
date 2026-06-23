package export

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ebitdock/internal/build"
	"ebitdock/internal/config"
	"ebitdock/internal/process"
)

func Web(ctx context.Context, root string, cfg config.Config) error {
	status := process.NewStatus(cfg)
	result := build.WASM(ctx, root, cfg, status)
	fmt.Print(result.Log)
	if result.Err != nil {
		return result.Err
	}
	dist := filepath.Join(root, "dist")
	_ = os.RemoveAll(dist)
	if err := copyDir(filepath.Join(root, cfg.StaticRoot()), dist); err != nil {
		return err
	}
	fmt.Printf("exported web build to %s\n", dist)
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
