package seed

import (
	"os"
	"path/filepath"
	"testing"
)

func newRoots(t *testing.T) map[string]string {
	t.Helper()
	base := t.TempDir()
	return map[string]string{
		"memory": filepath.Join(base, "memory"),
		"skills": filepath.Join(base, "skills"),
	}
}

func mdFiles(root string) map[string]bool {
	t := map[string]bool{}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			t[p] = true
		}
		return nil
	})
	return t
}

func TestBootstrapFresh(t *testing.T) {
	roots := newRoots(t)
	changes, err := Bootstrap(roots)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if len(changes) < 2 {
		t.Fatalf("expect changes for both roots, got %v", changes)
	}
	for name, want := range map[string]string{
		filepath.Join(roots["memory"], "MEMORY.md"):            "MEMORY",
		filepath.Join(roots["memory"], "SOUL.md"):              "SOUL",
		filepath.Join(roots["memory"], "USER.md"):              "USER",
		filepath.Join(roots["memory"], "meta", "memory_index.md"): "index",
	} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("missing seeded file %s: %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("empty seeded file %s", name)
		}
		_ = want
	}
	stamp, err := os.ReadFile(filepath.Join(roots["memory"], stampName))
	if err != nil {
		t.Fatalf("missing seed stamp: %v", err)
	}
	if string(stamp) != SeedVersion+"\n" {
		t.Fatalf("stamp = %q, want %q", stamp, SeedVersion+"\n")
	}
	// skills 至少 10 个技能且各带 SKILL.md
	entries, err := os.ReadDir(roots["skills"])
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(roots["skills"], e.Name(), "SKILL.md")); err != nil {
				t.Fatalf("skill %s has no SKILL.md", e.Name())
			}
			n++
		}
	}
	if n < 10 {
		t.Fatalf("expected >=10 seeded skills, got %d", n)
	}
}

func TestBootstrapIdempotent(t *testing.T) {
	roots := newRoots(t)
	if _, err := Bootstrap(roots); err != nil {
		t.Fatal(err)
	}
	before := mdFiles(roots["memory"])
	after := mdFiles(roots["skills"])
	changes, err := Bootstrap(roots)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range changes {
		if !contains(c, "seed skipped") {
			t.Fatalf("second run must skip seeding, got change %q", c)
		}
	}
	if !sameSet(before, mdFiles(roots["memory"])) || !sameSet(after, mdFiles(roots["skills"])) {
		t.Fatal("second bootstrap must not change any file")
	}
}

func TestBootstrapZeroOverwrite(t *testing.T) {
	roots := newRoots(t)
	userFile := filepath.Join(roots["memory"], "my-note.md")
	if err := os.MkdirAll(roots["memory"], 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	changes, err := Bootstrap(roots)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(userFile)
	if err != nil || string(data) != "keep me" {
		t.Fatalf("user file overwritten or lost: %v %q", err, data)
	}
	if _, err := os.Stat(filepath.Join(roots["memory"], "MEMORY.md")); err == nil {
		t.Fatal("non-empty root must not be seeded")
	}
	found := false
	for _, c := range changes {
		if contains(c, "zero-overwrite") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected zero-overwrite note, got %v", changes)
	}
}

func TestBootstrapLinkSkipped(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "real")
	root := filepath.Join(base, "link")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, root); err != nil {
		t.Skip("symlink unavailable on this environment:", err)
	}
	changes, err := Bootstrap(map[string]string{"memory": root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "MEMORY.md")); err == nil {
		t.Fatal("must never write through a link")
	}
	if len(changes) != 1 || !contains(changes[0], "left untouched") {
		t.Fatalf("unexpected changes %v", changes)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (stringIndex(s, sub) >= 0))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}
