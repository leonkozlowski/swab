// Package diff renders minimal unified diffs for --diff output.
package diff

import (
	"fmt"
	"strings"
)

const context = 3

type op struct {
	kind byte // ' ', '-', '+'
	a, b int  // line positions in old/new at this op
	line string
}

// Unified returns a unified diff of a -> b labelled with name, or "" if equal.
func Unified(name string, a, b []byte) string {
	if string(a) == string(b) {
		return ""
	}
	ops := lineOps(splitLines(string(a)), splitLines(string(b)))

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", name, name)
	for i := 0; i < len(ops); {
		if ops[i].kind == ' ' {
			i++
			continue
		}
		start := max(0, i-context)
		end := i
		for {
			for end < len(ops) && ops[end].kind != ' ' {
				end++
			}
			k := end
			for k < len(ops) && ops[k].kind == ' ' {
				k++
			}
			if k < len(ops) && k-end <= 2*context {
				end = k
				continue
			}
			end = min(len(ops), end+context)
			break
		}
		writeHunk(&sb, ops[start:end])
		i = end
	}
	return sb.String()
}

func writeHunk(sb *strings.Builder, h []op) {
	aLen, bLen := 0, 0
	for _, o := range h {
		if o.kind != '+' {
			aLen++
		}
		if o.kind != '-' {
			bLen++
		}
	}
	aStart, bStart := h[0].a+1, h[0].b+1
	if aLen == 0 {
		aStart--
	}
	if bLen == 0 {
		bStart--
	}
	fmt.Fprintf(sb, "@@ -%d,%d +%d,%d @@\n", aStart, aLen, bStart, bLen)
	for _, o := range h {
		sb.WriteByte(o.kind)
		sb.WriteString(o.line)
		sb.WriteByte('\n')
	}
}

// lineOps computes an LCS edit script; huge inputs fall back to replace-all.
func lineOps(a, b []string) []op {
	n, m := len(a), len(b)
	if n*m > 25_000_000 {
		var ops []op
		for i, l := range a {
			ops = append(ops, op{'-', i, 0, l})
		}
		for j, l := range b {
			ops = append(ops, op{'+', n, j, l})
		}
		return ops
	}
	lcs := make([][]int32, n+1)
	for i := range lcs {
		lcs[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var ops []op
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, op{' ', i, j, a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, op{'-', i, j, a[i]})
			i++
		default:
			ops = append(ops, op{'+', i, j, b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{'-', i, j, a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, op{'+', i, j, b[j]})
	}
	return ops
}

const noNewline = "\n\\ No newline at end of file"

// splitLines tags a final unterminated line with the marker so it diffs and prints correctly.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	if !strings.HasSuffix(s, "\n") {
		lines[len(lines)-1] += noNewline
	}
	return lines
}
