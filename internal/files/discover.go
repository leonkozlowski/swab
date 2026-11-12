// Package files finds Helm template files to format.
package files

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var templateExts = map[string]bool{".yaml": true, ".yml": true, ".tpl": true, ".txt": true}

// Discover returns template files under templates/ dirs next to a Chart.yaml, or root itself if it is a file.
func Discover(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	var out []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if templateExts[filepath.Ext(p)] && inTemplatesDir(p) {
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

func inTemplatesDir(p string) bool {
	dir := filepath.Dir(p)
	for {
		if filepath.Base(dir) == "templates" {
			if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "Chart.yaml")); err == nil {
				return true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}
