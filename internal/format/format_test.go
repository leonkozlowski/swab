package format

import (
	"errors"
	"testing"
)

func TestSource(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"pad action", `{{.Values.x}}`, `{{ .Values.x }}`},
		{"collapse spaces", `{{-   .x   -}}`, `{{- .x -}}`},
		{"pipes", `{{ .x|quote|indent 2 }}`, `{{ .x | quote | indent 2 }}`},
		{"declare", `{{ $x:=1 }}`, `{{ $x := 1 }}`},
		{"assign", `{{ $x := 1 }}{{ $x   =   2 }}`, `{{ $x := 1 }}{{ $x = 2 }}`},
		{"range vars", `{{ range $i,$v := .l }}{{ end }}`, `{{ range $i, $v := .l }}{{ end }}`},
		{"parens", `{{ ( .x ) }}`, `{{ (.x) }}`},
		{"field on parens", `{{ (index .Values "a").b }}`, `{{ (index .Values "a").b }}`},
		{"braces in string", `{{ "}}|x" }}`, `{{ "}}|x" }}`},
		{"raw string", "{{ `a  |  b` }}", "{{ `a  |  b` }}"},
		{"negative number not trim", `{{-3}}`, `{{ -3 }}`},
		{"comment untouched", `{{/*  a   b  */}}`, `{{/*  a   b  */}}`},
		{"trim comment untouched", `{{- /*  a  */ -}}`, `{{- /*  a  */ -}}`},
		{"multiline untouched", "{{ dict\n  \"a\" 1 }}", "{{ dict\n  \"a\" 1 }}"},
		{"yaml untouched", "a:   b\n  c: {{.x}}\n", "a:   b\n  c: {{ .x }}\n"},
		{"define", "{{- define \"x\" -}}\n{{.y}}\n{{- end}}", "{{- define \"x\" -}}\n{{ .y }}\n{{- end }}"},
		{"helpers", `{{ include "demo.labels" .|nindent 4 }}`, `{{ include "demo.labels" . | nindent 4 }}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Source("t", []byte(tt.in))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("\n got: %q\nwant: %q", got, tt.want)
			}
			again, err := Source("t", got)
			if err != nil || string(again) != string(got) {
				t.Fatalf("not idempotent: %q -> %q (err %v)", got, again, err)
			}
		})
	}
}

func TestSyntaxError(t *testing.T) {
	_, err := Source("t", []byte(`{{ if .x }}unclosed`))
	var se *SyntaxError
	if !errors.As(err, &se) {
		t.Fatalf("want SyntaxError, got %v", err)
	}
}

func TestSplitRoundTrip(t *testing.T) {
	inputs := []string{"", "plain", "{{ x", "a {{ .b }} c {{- .d -}} e", "{{ \"}}\" }}tail"}
	for _, in := range inputs {
		var s string
		for _, seg := range split(in) {
			s += seg.raw
		}
		if s != in {
			t.Errorf("split(%q) round-trip = %q", in, s)
		}
	}
}
