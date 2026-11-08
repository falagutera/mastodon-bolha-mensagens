package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dump writes a page rendered by a test to the directory in JUSTMYTOOTS_DUMP, with
// the assets next to it, so it can be opened in a browser. Without JUSTMYTOOTS_DUMP
// it does nothing.
func dump(t *testing.T, name, body string) {
	t.Helper()
	dir := os.Getenv("JUSTMYTOOTS_DUMP")
	if dir == "" {
		return
	}
	err := fs.WalkDir(assetFS, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := assetFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, path), b, 0o644)
	})
	if err == nil {
		body = strings.ReplaceAll(body, `"/assets/`, `"assets/`)
		err = os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)
	}
	if err != nil {
		t.Fatal(err)
	}
}
