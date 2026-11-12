package format

import (
	"fmt"
	"text/template/parse"
)

// parseTrees parses src like text/template, without needing Helm's funcs.
func parseTrees(name, src string) (map[string]*parse.Tree, error) {
	trees := map[string]*parse.Tree{}
	t := parse.New(name)
	t.Mode = parse.ParseComments | parse.SkipFuncCheck
	if _, err := t.Parse(src, "", "", trees); err != nil {
		return nil, err
	}
	return trees, nil
}

// equivalent reports whether out parses to the same trees as want (text compared post-trim).
func equivalent(name string, want map[string]*parse.Tree, out string) error {
	got, err := parseTrees(name, out)
	if err != nil {
		return fmt.Errorf("formatted output does not parse: %w", err)
	}
	if len(got) != len(want) {
		return fmt.Errorf("template count changed: %d -> %d", len(want), len(got))
	}
	for n, w := range want {
		g, ok := got[n]
		if !ok {
			return fmt.Errorf("template %q disappeared", n)
		}
		if w.Root.String() != g.Root.String() {
			return fmt.Errorf("template %q changed", n)
		}
	}
	return nil
}
