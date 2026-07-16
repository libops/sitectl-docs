package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortableText(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("user home is unavailable")
	}

	tests := map[string]string{
		home:                                  "~",
		filepath.Join(home, ".ssh", "id_rsa"): "~/.ssh/id_rsa",
		"Example: " + filepath.Join(home, ".ssh", "id_ed25519"): "Example: ~/.ssh/id_ed25519",
		"/var/lib/libops/config":                                "/var/lib/libops/config",
	}
	for input, want := range tests {
		if got := portableText(input); got != want {
			t.Errorf("portableText(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemoveStaleGeneratedSnippets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	current := filepath.Join(dir, "current.mdx")
	stale := filepath.Join(dir, "stale.mdx")
	manual := filepath.Join(dir, "manual.mdx")
	legacy := filepath.Join(dir, "sitectl-isle-sync.mdx")
	for path, contents := range map[string]string{
		current: autoGenHeader + "current\n",
		stale:   autoGenHeader + "stale\n",
		manual:  "manually maintained\n",
		legacy:  "{/* Legacy snapshot from sitectl-isle v0.19.0. */}\n",
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
	for _, path := range []string{current, manual, legacy} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
}
