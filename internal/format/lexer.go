package format

import "strings"

type segKind int

const (
	segText segKind = iota
	segAction
)

// segment is either literal text or a single {{ ... }} action.
type segment struct {
	kind      segKind
	pos       int    // byte offset of the segment in the source
	raw       string // exact original source of the segment
	leftTrim  bool   // action opened with "{{- "
	rightTrim bool   // action closed with " -}}"
	body      string // action contents between delimiters and trim markers
}

// split cuts src into text and action segments whose raw fields rejoin to src.
func split(src string) []segment {
	var segs []segment
	textStart, i := 0, 0
	for {
		k := strings.Index(src[i:], "{{")
		if k < 0 {
			break
		}
		start := i + k
		end, seg, ok := scanAction(src, start)
		if !ok {
			break
		}
		if start > textStart {
			segs = append(segs, segment{kind: segText, pos: textStart, raw: src[textStart:start]})
		}
		segs = append(segs, seg)
		i, textStart = end, end
	}
	if textStart < len(src) {
		segs = append(segs, segment{kind: segText, pos: textStart, raw: src[textStart:]})
	}
	return segs
}

// scanAction scans an action like text/template's lexer, skipping literals and comments.
func scanAction(src string, start int) (end int, seg segment, ok bool) {
	seg.kind = segAction
	seg.pos = start
	j := start + 2
	if j+1 < len(src) && src[j] == '-' && isSpace(src[j+1]) {
		seg.leftTrim = true
		j++
	}
	bodyStart := j
	for j < len(src) {
		c := src[j]
		switch {
		case c == '"' || c == '\'':
			j = skipQuoted(src, j)
			if j < 0 {
				return 0, seg, false
			}
		case c == '`':
			n := strings.IndexByte(src[j+1:], '`')
			if n < 0 {
				return 0, seg, false
			}
			j += n + 2
		case c == '/' && j+1 < len(src) && src[j+1] == '*':
			n := strings.Index(src[j+2:], "*/")
			if n < 0 {
				return 0, seg, false
			}
			j += n + 4
		case c == '}' && j+1 < len(src) && src[j+1] == '}':
			bodyEnd := j
			if bodyEnd-2 >= bodyStart && src[bodyEnd-1] == '-' && isSpace(src[bodyEnd-2]) {
				seg.rightTrim = true
				bodyEnd--
			}
			seg.body = src[bodyStart:bodyEnd]
			seg.raw = src[start : j+2]
			return j + 2, seg, true
		default:
			j++
		}
	}
	return 0, seg, false
}

// skipQuoted returns the index past the quoted literal at s[i], or -1.
func skipQuoted(s string, i int) int {
	q := s[i]
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case q:
			return j + 1
		}
	}
	return -1
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}
