package main

import (
	"os"
	"path/filepath"
	"testing"
)

// docs/ is the schemas' and examples': go generate ./... after a change.
func TestDocs(t *testing.T) {
	out := t.TempDir()
	if err := run(out, "../../../examples"); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(out, path)
		want, _ := os.ReadFile(path) //#nosec G122 G304 -- the test's own temporary directory
		have, err := os.ReadFile(filepath.Join("../../../docs", rel))
		if err != nil || string(have) != string(want) {
			t.Errorf("docs/%s is not the schemas': go generate ./...", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
