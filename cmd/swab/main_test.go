package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRenderEquivalence requires identical `helm template` output before and after formatting.
func TestRenderEquivalence(t *testing.T) {
	charts, err := filepath.Glob("../../testdata/charts/*")
	if err != nil || len(charts) == 0 {
		t.Fatalf("no test charts found: %v", err)
	}
	helm, helmErr := exec.LookPath("helm")

	for _, src := range charts {
		t.Run(filepath.Base(src), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), filepath.Base(src))
			if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
				t.Fatal(err)
			}

			var before string
			if helmErr == nil {
				before = render(t, helm, dir)
			}

			var stderr bytes.Buffer
			if code := run([]string{"fmt", dir}, nil, io.Discard, &stderr); code != 0 {
				t.Fatalf("swab fmt exited %d: %s", code, stderr.String())
			}

			var out bytes.Buffer
			if code := run([]string{"fmt", "--check", dir}, nil, &out, &stderr); code != 0 {
				t.Fatalf("second pass not idempotent, would change:\n%s", out.String())
			}

			if helmErr != nil {
				t.Skip("helm not on PATH; skipped render comparison")
			}
			if after := render(t, helm, dir); after != before {
				t.Fatalf("rendered output changed after formatting\n--- before\n%s\n--- after\n%s", before, after)
			}
		})
	}
}

func render(t *testing.T, helm, dir string) string {
	t.Helper()
	cmd := exec.Command(helm, "template", "rel", dir)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("helm template failed: %v\n%s", err, errb.String())
	}
	return out.String()
}

func TestCheckReportsUnformatted(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"fmt", "--check", "../../testdata/charts/demo"}, nil, &out, &errb)
	if code != 1 || !strings.Contains(out.String(), "deployment.yaml") {
		t.Fatalf("want exit 1 listing deployment.yaml, got %d: %q %q", code, out.String(), errb.String())
	}
}

func TestStdin(t *testing.T) {
	tests := []struct {
		name, in, wantOut string
		args              []string
		wantCode          int
	}{
		{"format", "{{.x}}\n", "{{ .x }}\n", []string{"fmt", "-"}, 0},
		{"check changed", "{{.x}}\n", "<stdin>\n", []string{"fmt", "--check", "-"}, 1},
		{"check clean", "{{ .x }}\n", "", []string{"fmt", "--check", "-"}, 0},
		{"diff", "{{.x}}\n", "--- <stdin>\n+++ <stdin>\n@@ -1,1 +1,1 @@\n-{{.x}}\n+{{ .x }}\n", []string{"fmt", "--diff", "-"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.in), &out, &errb)
			if code != tt.wantCode || out.String() != tt.wantOut {
				t.Fatalf("got exit %d out %q (stderr %q), want exit %d out %q", code, out.String(), errb.String(), tt.wantCode, tt.wantOut)
			}
		})
	}
}

// TestFailedWriteKeepsOriginal makes the chart dir read-only so the temp file can't be created.
func TestFailedWriteKeepsOriginal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not block writes on Windows")
	}
	dir := t.TempDir()
	tpl := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tpl, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "Chart.yaml"), "apiVersion: v2\nname: x\nversion: 0.1.0\n")
	file := filepath.Join(tpl, "cm.yaml")
	orig := []byte("a: {{.x}}\n")
	writeFile(t, file, string(orig))
	if err := os.Chmod(tpl, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tpl, 0o755) })

	var errb bytes.Buffer
	if code := run([]string{"fmt", dir}, nil, io.Discard, &errb); code != 1 {
		t.Fatalf("want exit 1, got %d (%s)", code, errb.String())
	}
	if got, _ := os.ReadFile(file); !bytes.Equal(got, orig) {
		t.Fatalf("original modified after failed write: %q", got)
	}
}

func TestWriteAtomicKeepsModeAndSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need elevated privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "real.yaml")
	link := filepath.Join(dir, "link.yaml")
	writeFile(t, target, "old")
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(link, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced by a regular file")
	}
	if got, _ := os.ReadFile(target); string(got) != "new" {
		t.Fatalf("target = %q, want new", got)
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", fi.Mode().Perm())
	}
	if leftovers, _ := filepath.Glob(filepath.Join(dir, ".*.swab-*")); len(leftovers) > 0 {
		t.Fatalf("temp files left behind: %v", leftovers)
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
