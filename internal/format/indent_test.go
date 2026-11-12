package format

import (
	"strings"
	"testing"
)

// TestCRLF checks CRLF input formats exactly like LF input, keeping CRLF endings.
func TestCRLF(t *testing.T) {
	lf := "{{- define \"x\" -}}\n{{- if .a }}\n{{- .b|quote }}\n{{- end }}\n{{- end }}\nk: {{.v}}\n"
	want := "{{- define \"x\" -}}\n  {{- if .a }}\n    {{- .b | quote }}\n  {{- end }}\n{{- end }}\nk: {{ .v }}\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	got, err := Source("t", []byte(crlf))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != strings.ReplaceAll(want, "\n", "\r\n") {
		t.Fatalf("\n got: %q\nwant: %q", got, strings.ReplaceAll(want, "\n", "\r\n"))
	}
}

func TestReindent(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			name: "helpers nest like code",
			in: `{{- define "x.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 }}
{{- end }}
{{- end }}
{{- end }}
`,
			want: `{{- define "x.fullname" -}}
  {{- if .Values.fullnameOverride }}
    {{- .Values.fullnameOverride | trunc 63 }}
  {{- else }}
    {{- $name := default .Chart.Name .Values.nameOverride }}
    {{- if contains $name .Release.Name }}
      {{- .Release.Name | trunc 63 }}
    {{- end }}
  {{- end }}
{{- end }}
`,
		},
		{
			name: "control aligns with the YAML it wraps",
			in: `spec:
{{- with .Values.env }}
  env:
        {{- range $k, $v := . }}
    - name: {{ $k }}
      value: {{ $v }}
 {{- end }}
      {{- end }}
`,
			want: `spec:
  {{- with .Values.env }}
  env:
    {{- range $k, $v := . }}
    - name: {{ $k }}
      value: {{ $v }}
    {{- end }}
  {{- end }}
`,
		},
		{
			name: "output lines in YAML keep their position",
			in: `labels:
  {{- with .Values.podLabels }}
  {{- toYaml . | nindent 2 }}
  {{- end }}
`,
			want: `labels:
  {{- with .Values.podLabels }}
  {{- toYaml . | nindent 2 }}
  {{- end }}
`,
		},
		{
			name: "variable declaration aligns with following YAML",
			in: `{{- range .Values.items }}
{{- $n := .name }}
- name: {{ $n }}
{{- end }}
`,
			want: `{{- range .Values.items }}
{{- $n := .name }}
- name: {{ $n }}
{{- end }}
`,
		},
		{
			name: "untrimmed control lines are never moved",
			in: `a:
{{ if .x }}
  b: 1
{{ end }}
`,
			want: `a:
{{ if .x }}
  b: 1
{{ end }}
`,
		},
		{
			name: "define with YAML body stays at column 0",
			in: `  {{- define "x.labels" -}}
app: x
{{- if .Chart.AppVersion }}
version: {{ .Chart.AppVersion }}
    {{- end }}
{{- end }}
`,
			want: `{{- define "x.labels" -}}
app: x
{{- if .Chart.AppVersion }}
version: {{ .Chart.AppVersion }}
{{- end }}
{{- end }}
`,
		},
		{
			name: "YAML in a nested block keeps the define in code mode",
			in: `{{- define "x.pullSecrets" -}}
{{- $s := list }}
{{- range .images -}}
{{- $s = append $s .name -}}
{{- end -}}
{{- if $s -}}
imagePullSecrets:
    {{- range $s }}
  - name: {{ . }}
    {{- end }}
{{- end }}
{{- end -}}
`,
			want: `{{- define "x.pullSecrets" -}}
  {{- $s := list }}
  {{- range .images -}}
    {{- $s = append $s .name -}}
  {{- end -}}
  {{- if $s -}}
imagePullSecrets:
  {{- range $s }}
  - name: {{ . }}
  {{- end }}
  {{- end }}
{{- end -}}
`,
		},
		{
			name: "flush toYaml lines do not set alignment",
			in: `        - matchLabels:
            app: x
            {{- if .Values.podLabels }}
{{ toYaml .Values.podLabels | indent 12 }}
            {{- end }}
`,
			want: `        - matchLabels:
            app: x
            {{- if .Values.podLabels }}
{{ toYaml .Values.podLabels | indent 12 }}
            {{- end }}
`,
		},
		{
			name: "declarations in code-nested output blocks follow code indentation",
			in: `{{- define "x" -}}
{{- $seen := list }}
{{- range .items }}
{{- if not (has . $seen) }}
{{- $seen = append $seen . }}
{{ include "x.item" . }}
{{- end }}
{{- end }}
{{- end -}}
`,
			want: `{{- define "x" -}}
  {{- $seen := list }}
  {{- range .items }}
    {{- if not (has . $seen) }}
      {{- $seen = append $seen . }}
{{ include "x.item" . }}
    {{- end }}
  {{- end }}
{{- end -}}
`,
		},
		{
			name: "multi-line actions are left alone",
			in: `{{- define "x" -}}
{{- $d := dict
    "a" 1 }}
{{- end }}
`,
			want: `{{- define "x" -}}
{{- $d := dict
    "a" 1 }}
{{- end }}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Source("t", []byte(tt.in))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("\n--- got\n%s\n--- want\n%s", got, tt.want)
			}
			again, err := Source("t", got)
			if err != nil || string(again) != string(got) {
				t.Fatalf("not idempotent:\n%s\n--- second pass\n%s (err %v)", got, again, err)
			}
		})
	}
}
