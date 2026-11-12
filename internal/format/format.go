// Package format formats Helm templates and verifies the parse tree is unchanged.
package format

import (
	"errors"
	"fmt"
	"strings"
)

// SyntaxError reports that the input template does not parse.
type SyntaxError struct{ Err error }

func (e *SyntaxError) Error() string { return "template syntax error: " + e.Err.Error() }
func (e *SyntaxError) Unwrap() error { return e.Err }

// ErrUnsafe means formatting would change semantics (a swab bug).
var ErrUnsafe = errors.New("formatting would change template semantics (this is a swab bug, please report it)")

// Source formats a single template file. name is used in error messages.
func Source(name string, src []byte) ([]byte, error) {
	in := string(src)
	want, err := parseTrees(name, in)
	if err != nil {
		return nil, &SyntaxError{Err: err}
	}

	var b strings.Builder
	b.Grow(len(in))
	for _, s := range split(in) {
		if s.kind == segText {
			b.WriteString(s.raw)
			continue
		}
		b.WriteString(formatAction(s))
	}
	out := reindent(b.String())

	if out != in {
		if err := equivalent(name, want, out); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnsafe, err)
		}
	}
	return []byte(out), nil
}

// formatAction normalizes spacing; comments and multi-line actions are untouched.
func formatAction(s segment) string {
	if strings.ContainsAny(s.body, "\r\n") {
		return s.raw
	}
	trimmed := strings.TrimSpace(s.body)
	if trimmed == "" || strings.HasPrefix(trimmed, "/*") {
		return s.raw
	}
	toks, ok := tokenize(s.body)
	if !ok || len(toks) == 0 {
		return s.raw
	}

	var b strings.Builder
	b.WriteString("{{")
	if s.leftTrim {
		b.WriteByte('-')
	}
	b.WriteByte(' ')
	for i, t := range toks {
		if i > 0 && needSpace(toks[i-1], t) {
			b.WriteByte(' ')
		}
		b.WriteString(t.text)
	}
	b.WriteByte(' ')
	if s.rightTrim {
		b.WriteByte('-')
	}
	b.WriteString("}}")
	return b.String()
}

type tokKind int

const (
	tokWord tokKind = iota // identifiers, fields, numbers, strings, ...
	tokLParen
	tokRParen
	tokPipe
	tokComma
	tokAssign // := or =
)

type token struct {
	text        string
	kind        tokKind
	spaceBefore bool // whitespace preceded this token in the original
}

func tokenize(body string) ([]token, bool) {
	var toks []token
	space := false
	emit := func(text string, kind tokKind) {
		toks = append(toks, token{text: text, kind: kind, spaceBefore: space})
		space = false
	}
	for i := 0; i < len(body); {
		c := body[i]
		switch {
		case isSpace(c):
			space = true
			i++
		case c == '"' || c == '\'':
			j := skipQuoted(body, i)
			if j < 0 {
				return nil, false
			}
			emit(body[i:j], tokWord)
			i = j
		case c == '`':
			n := strings.IndexByte(body[i+1:], '`')
			if n < 0 {
				return nil, false
			}
			emit(body[i:i+n+2], tokWord)
			i += n + 2
		case c == '(':
			emit("(", tokLParen)
			i++
		case c == ')':
			emit(")", tokRParen)
			i++
		case c == '|':
			emit("|", tokPipe)
			i++
		case c == ',':
			emit(",", tokComma)
			i++
		case c == ':' && i+1 < len(body) && body[i+1] == '=':
			emit(":=", tokAssign)
			i += 2
		case c == '=':
			emit("=", tokAssign)
			i++
		default:
			j := i
			for j < len(body) && !isWordBreak(body, j) {
				j++
			}
			if j == i { // lone ':' or similar; let it be its own word
				j++
			}
			emit(body[i:j], tokWord)
			i = j
		}
	}
	return toks, true
}

func isWordBreak(s string, j int) bool {
	c := s[j]
	if isSpace(c) || strings.IndexByte("()|,=\"'`", c) >= 0 {
		return true
	}
	return c == ':' && j+1 < len(s) && s[j+1] == '='
}

func needSpace(prev, cur token) bool {
	switch {
	case prev.kind == tokLParen, cur.kind == tokRParen, cur.kind == tokComma:
		return false
	case prev.kind == tokComma,
		prev.kind == tokPipe, cur.kind == tokPipe,
		prev.kind == tokAssign, cur.kind == tokAssign:
		return true
	default:
		// Preserve adjacency such as `(index .x "a").b`.
		return cur.spaceBefore
	}
}
