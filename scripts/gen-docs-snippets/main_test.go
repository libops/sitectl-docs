package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveStaleGeneratedSnippets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	current := filepath.Join(dir, "current.mdx")
	stale := filepath.Join(dir, "stale.mdx")
	manual := filepath.Join(dir, "manual.mdx")
	for path, contents := range map[string]string{
		current: autoGenHeader + "current\n",
		stale:   autoGenHeader + "stale\n",
		manual:  "manually maintained\n",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	if err := removeStaleGeneratedSnippets(dir, map[string]struct{}{current: {}}); err != nil {
		t.Fatalf("remove stale snippets: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale generated snippet still exists: %v", err)
	}
	for _, path := range []string{current, manual} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
}
