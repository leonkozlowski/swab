<div align="center">

# swab

A formatter for Helm charts.

[![Release](https://img.shields.io/github/v/release/leonkozlowski/swab?sort=semver)](https://github.com/leonkozlowski/swab/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![CI: Test](https://github.com/leonkozlowski/swab/actions/workflows/test.yml/badge.svg)](https://github.com/leonkozlowski/swab/actions/workflows/test.yml)
[![CI: Lint](https://github.com/leonkozlowski/swab/actions/workflows/lint.yml/badge.svg)](https://github.com/leonkozlowski/swab/actions/workflows/lint.yml)

</div>

```diff
-  name: {{include "demo.fullname" .}}
+  name: {{ include "demo.fullname" . }}
   labels:
-    {{- include "demo.labels" .|nindent 4}}
+    {{- include "demo.labels" . | nindent 4 }}
```

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/leonkozlowski/swab/main/install.sh | sh
```

Or with Go:

```sh
go install github.com/leonkozlowski/swab/cmd/swab@latest
```

Binaries for macOS, Linux and Windows are on the [releases page](https://github.com/leonkozlowski/swab/releases).

## Usage

```sh
swab fmt                # format charts under the current directory
swab fmt ./charts/api   # format one chart
swab fmt --check .      # exit 1 if anything needs formatting
swab fmt --diff .       # show changes without writing
swab fmt - < file.yaml  # read stdin, write stdout
```

swab formats files in `templates/` next to a `Chart.yaml`.

## What it does

- Adds spaces inside actions: `{{.x}}` becomes `{{ .x }}`
- Adds spaces around pipes and assignments: `.x|quote` becomes `.x | quote`
- Indents `if`, `range`, `with` and `define` blocks
- Leaves YAML, comments and multi-line actions alone

swab only re-indents lines that start with `{{-`, since Helm trims that whitespace anyway.

## How it stays safe

swab parses each template before and after formatting and compares the results.
If they differ, it leaves the file alone.

The tests also run `helm template` before and after formatting and check the output is identical.

## License

[MIT](LICENSE)
