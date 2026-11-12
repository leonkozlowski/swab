package diff

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUnifiedNoNewlineMarker(t *testing.T) {
	got := Unified("f", []byte("a\n{{.x}}"), []byte("a\n{{ .x }}"))
	want := "--- f\n+++ f\n@@ -1,2 +1,2 @@\n a\n-{{.x}}\n\\ No newline at end of file\n+{{ .x }}\n\\ No newline at end of file\n"
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestUnifiedNewlineOnlyChange(t *testing.T) {
	got := Unified("f", []byte("a"), []byte("a\n"))
	if !strings.Contains(got, "-a\n\\ No newline at end of file\n+a\n") {
		t.Fatalf("newline-only change not shown: %q", got)
	}
}

// TestUnifiedApplies checks generated diffs apply cleanly with patch(1).
func TestUnifiedApplies(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the runner's patch 2.5.9 crashes on no-newline markers")
	}
	patch, err := exec.LookPath("patch")
	if err != nil {
		t.Skip("patch not on PATH")
	}
	cases := [][2]string{
		{"a\n{{.x}}", "a\n{{ .x }}"},
		{"a\nb", "a\nb\n"},
		{"a\nb\n", "a\nb"},
		{"1\n2\n3\n4\n5\n6\n7\n8\n9\n{{.x}}\n", "1\n2\n3\n4\n5\n6\n7\n8\n9\n{{ .x }}\n"},
		{"{{.a}}\n1\n2\n3\n4\n5\n6\n7\n8\n{{.b}}", "{{ .a }}\n1\n2\n3\n4\n5\n6\n7\n8\n{{ .b }}"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		file, diffFile := filepath.Join(dir, "f"), filepath.Join(dir, "f.diff")
		if err := os.WriteFile(file, []byte(c[0]), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(diffFile, []byte(Unified("f", []byte(c[0]), []byte(c[1]))), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(patch, "-s", file, diffFile).CombinedOutput(); err != nil {
			t.Fatalf("patch failed for %q -> %q: %v\n%s", c[0], c[1], err, out)
		}
		if got, _ := os.ReadFile(file); string(got) != c[1] {
			t.Fatalf("patched %q, want %q", got, c[1])
		}
	}
}
