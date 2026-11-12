package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/leonkozlowski/swab/internal/diff"
	"github.com/leonkozlowski/swab/internal/files"
	"github.com/leonkozlowski/swab/internal/format"
)

type fmtOptions struct {
	check bool
	diff  bool
}

func newFmtCmd() *cobra.Command {
	var opts fmtOptions
	cmd := &cobra.Command{
		Use:   "fmt [path ...]",
		Short: "Format Helm chart templates in place",
		Long:  "Format templates under templates/ next to each Chart.yaml found in the given paths (default: .).\nUse \"-\" as the only path to read stdin and write stdout.",
		Example: `  swab fmt
  swab fmt ./charts/api
  swab fmt --check .
  swab fmt --diff .
  swab fmt - < deployment.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFmt(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.check, "check", "c", false, "list files that need formatting and exit 1 if any; write nothing")
	cmd.Flags().BoolVarP(&opts.diff, "diff", "d", false, "print a diff of the changes; write nothing")
	return cmd
}

func runFmt(stdin io.Reader, stdout, stderr io.Writer, paths []string, opts fmtOptions) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	if len(paths) == 1 && paths[0] == "-" {
		return fmtStdin(stdin, stdout, stderr, opts)
	}

	var targets []string
	for _, p := range paths {
		found, err := files.Discover(p)
		if err != nil {
			return err
		}
		targets = append(targets, found...)
	}
	if len(targets) == 0 {
		warnf(stderr, "no Helm templates found (looked for templates/ next to Chart.yaml)")
		return nil
	}

	failed, changed := 0, 0
	for _, f := range targets {
		did, err := fmtFile(f, stdout, opts)
		if err != nil {
			errorf(stderr, "%s: %v", f, err)
			failed++
			continue
		}
		if did {
			changed++
		}
	}

	switch {
	case opts.check && changed > 0:
		warnf(stderr, "%d of %d files need formatting", changed, len(targets))
	case opts.check || opts.diff:
		if changed == 0 && failed == 0 {
			successf(stderr, "%d files already formatted", len(targets))
		}
	case changed > 0:
		successf(stderr, "formatted %d of %d files", changed, len(targets))
	case failed == 0:
		successf(stderr, "%d files already formatted", len(targets))
	}

	if failed > 0 || (opts.check && changed > 0) {
		return exitError(1)
	}
	return nil
}

// fmtFile formats one file and reports whether it changed (or would change).
func fmtFile(path string, stdout io.Writer, opts fmtOptions) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	out, err := format.Source(path, src)
	if err != nil {
		return false, err
	}
	if bytes.Equal(src, out) {
		return false, nil
	}
	switch {
	case opts.diff:
		return true, printDiff(stdout, diff.Unified(path, src, out))
	case opts.check:
		_, err = io.WriteString(stdout, path+"\n")
		return true, err
	default:
		return true, writeAtomic(path, out, info.Mode().Perm())
	}
}

// writeAtomic replaces path via a temp file and rename so a failed write never truncates the original.
func writeAtomic(path string, data []byte, perm os.FileMode) (err error) {
	if resolved, rerr := filepath.EvalSymlinks(path); rerr == nil {
		path = resolved
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".swab-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func fmtStdin(stdin io.Reader, stdout, stderr io.Writer, opts fmtOptions) error {
	const name = "<stdin>"
	src, err := io.ReadAll(stdin)
	if err != nil {
		return err
	}
	out, err := format.Source(name, src)
	if err != nil {
		errorf(stderr, "%s: %v", name, err)
		return exitError(1)
	}
	changed := !bytes.Equal(src, out)
	switch {
	case opts.diff:
		err = printDiff(stdout, diff.Unified(name, src, out))
	case opts.check && changed:
		_, err = io.WriteString(stdout, name+"\n")
	case !opts.check:
		_, err = stdout.Write(out)
	}
	if err != nil {
		return err
	}
	if opts.check && changed {
		return exitError(1)
	}
	return nil
}
