package format

import (
	"sort"
	"strings"
)

const indentWidth = 2

/*
Control-flow indentation.

Only lines holding a single one-line "{{-" action are moved; "{{-" trims the
preceding whitespace, so their indentation never reaches the output.

Blocks are laid out in one of two modes:
  - align: if/else/end match the least-indented YAML line inside the block;
    output lines like `{{- toYaml . | nindent 8 }}` are left alone.
  - code: pure-logic blocks inside a define nest two spaces per level.
*/

type lineKind int

const (
	lineBlank   lineKind = iota
	lineAnchor           // first visible thing is emitted text or an untrimmed action
	lineMovable          // whitespace + one single-line {{- action }} + whitespace
	lineOpaque           // inside a multi-line action, several actions, etc.
)

type block struct {
	keyword           string
	openSeg           int
	openLine, endLine int
	parent            *block
	skip              bool // opener and end on the same line; nothing to lay out
	code              bool // code mode (see above); otherwise align mode
	col, body         int
}

type role int

const (
	roleNone role = iota
	roleOpen
	roleMid
	roleClose
)

func reindent(src string) string {
	segs := split(src)

	starts := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	n := len(starts)
	lineEnd := func(l int) int {
		if l+1 < n {
			return starts[l+1] - 1
		}
		return len(src)
	}
	lineOf := func(off int) int {
		return sort.Search(n, func(i int) bool { return starts[i] > off }) - 1
	}
	indentOf := func(l int) int {
		ls, le := starts[l], lineEnd(l)
		p := ls
		for p < le && (src[p] == ' ' || src[p] == '\t') {
			p++
		}
		return p - ls
	}

	// Classify lines.
	actionAt := map[int]int{}
	opaque := make([]bool, n)
	for i, s := range segs {
		if s.kind != segAction {
			continue
		}
		actionAt[s.pos] = i
		if end := s.pos + len(s.raw); strings.Contains(s.raw, "\n") {
			for l := lineOf(s.pos) + 1; l < n && starts[l] < end; l++ {
				opaque[l] = true
			}
		}
	}
	kinds := make([]lineKind, n)
	lineSeg := make([]int, n)
	for l := 0; l < n; l++ {
		if opaque[l] {
			kinds[l] = lineOpaque
			continue
		}
		ls, le := starts[l], lineEnd(l)
		p := ls
		for p < le && isSpace(src[p]) {
			p++
		}
		if p == le {
			kinds[l] = lineBlank
			continue
		}
		si, ok := actionAt[p]
		if !ok || !segs[si].leftTrim {
			kinds[l] = lineAnchor
			continue
		}
		s := segs[si]
		if strings.Contains(s.raw, "\n") || strings.TrimSpace(src[s.pos+len(s.raw):le]) != "" {
			kinds[l] = lineOpaque
			continue
		}
		kinds[l] = lineMovable
		lineSeg[l] = si
	}

	// Build the block tree from every action, wherever it sits.
	var blocks []*block
	var stack []*block
	segBlock := map[int]*block{}
	segRole := map[int]role{}
	for i, s := range segs {
		if s.kind != segAction {
			continue
		}
		switch keyword(s.body) {
		case "if", "range", "with", "define", "block":
			b := &block{keyword: keyword(s.body), openSeg: i, openLine: lineOf(s.pos)}
			if len(stack) > 0 {
				b.parent = stack[len(stack)-1]
			}
			stack = append(stack, b)
			blocks = append(blocks, b)
			segBlock[i], segRole[i] = b, roleOpen
		case "else":
			if len(stack) == 0 {
				return src
			}
			segBlock[i], segRole[i] = stack[len(stack)-1], roleMid
		case "end":
			if len(stack) == 0 {
				return src
			}
			b := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			b.endLine = lineOf(s.pos)
			segBlock[i], segRole[i] = b, roleClose
		}
	}
	if len(stack) != 0 {
		return src
	}

	inner := make([]*block, n) // innermost block strictly containing each line
	for _, b := range blocks {
		b.skip = b.openLine == b.endLine
		if b.skip {
			continue
		}
		for l := b.openLine + 1; l < b.endLine; l++ {
			inner[l] = b
		}
	}
	nextAnchor := func(b *block, from int) int {
		for l := from + 1; l < b.endLine; l++ {
			if kinds[l] == lineAnchor {
				return l
			}
		}
		return -1
	}

	minAnchor := func(b *block) int {
		m := -1
		for l := b.openLine + 1; l < b.endLine; l++ {
			if kinds[l] == lineAnchor && (m < 0 || indentOf(l) < m) {
				m = indentOf(l)
			}
		}
		return m
	}

	// Lay out blocks, parents first (creation order).
	for _, b := range blocks {
		if b.skip {
			continue
		}
		anchor := minAnchor(b)
		b.code = anchor < 0 && (b.keyword == "define" || (b.parent != nil && b.parent.code))
		openerMovable := kinds[b.openLine] == lineMovable && lineSeg[b.openLine] == b.openSeg
		switch {
		case !openerMovable:
			b.col = indentOf(b.openLine)
		case b.keyword == "define":
			b.col = 0 // define is only legal at top level
		case anchor >= 0:
			b.col = anchor
		case b.code:
			b.col = b.parent.body
		default:
			b.col = indentOf(b.openLine)
		}
		b.body = b.col + indentWidth
	}

	// Decide each movable line's target indentation.
	target := make([]int, n)
	for l := range target {
		target[l] = -1
	}
	for l := 0; l < n; l++ {
		if kinds[l] != lineMovable {
			continue
		}
		si := lineSeg[l]
		if b := segBlock[si]; b != nil && segRole[si] != roleNone && !b.skip {
			target[l] = b.col
			continue
		}
		b := inner[l]
		switch {
		case b == nil:
			// top level: leave as written
		case b.code:
			target[l] = b.body
		case nonOutput(segs[si].body):
			// align declarations with the YAML that follows
			if a := nextAnchor(b, l); a >= 0 {
				target[l] = indentOf(a)
			} else {
				target[l] = b.col
			}
		default:
			// output lines in YAML keep their position
		}
	}

	var out strings.Builder
	out.Grow(len(src))
	for l := 0; l < n; l++ {
		line := src[starts[l]:lineEnd(l)]
		if target[l] >= 0 {
			line = strings.Repeat(" ", target[l]) + strings.TrimLeft(line, " \t")
		}
		out.WriteString(line)
		if l+1 < n {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// keyword returns the leading keyword of an action body, or "".
func keyword(body string) string {
	if strings.HasPrefix(strings.TrimSpace(body), "/*") {
		return ""
	}
	toks, ok := tokenize(body)
	if !ok || len(toks) == 0 {
		return ""
	}
	return toks[0].text
}

// nonOutput reports whether an action never writes output.
func nonOutput(body string) bool {
	if strings.HasPrefix(strings.TrimSpace(body), "/*") {
		return true
	}
	toks, ok := tokenize(body)
	if !ok || len(toks) == 0 {
		return false
	}
	if k := toks[0].text; k == "break" || k == "continue" {
		return true
	}
	return len(toks) >= 2 && strings.HasPrefix(toks[0].text, "$") && toks[1].kind == tokAssign
}
