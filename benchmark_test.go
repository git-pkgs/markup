package markup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readBenchdata(b *testing.B, name string) []byte {
	b.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		b.Fatalf("reading testdata/%s: %v", name, err)
	}
	return data
}

// --- Detect benchmarks ---

func BenchmarkDetect(b *testing.B) {
	filenames := []string{
		"README.md",
		"README.adoc",
		"README.rst.txt",
		"README.textile",
		"README.org",
		"README.unknown",
	}
	for b.Loop() {
		for _, f := range filenames {
			Detect(f)
		}
	}
}

// --- Markdown benchmarks ---

func BenchmarkMarkdownSmall(b *testing.B) {
	r := NewDefaultRegistry()
	content := readBenchdata(b, "README.markdown")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.md", content)
	}
}

func BenchmarkMarkdownMedium(b *testing.B) {
	r := NewDefaultRegistry()
	content := []byte(`# Project Name

A description of the project with **bold**, *italic*, and ` + "`code`" + ` formatting.

## Installation

` + "```bash" + `
go get github.com/example/project
` + "```" + `

## Usage

| Command | Description |
|---------|-------------|
| build   | Build the project |
| test    | Run tests |
| lint    | Run linters |

## Features

- [x] Feature one
- [x] Feature two
- [ ] Feature three
- [ ] Feature four

## Links

Visit https://example.com for more information.

See [the docs](https://docs.example.com) for API reference.

![logo](https://example.com/logo.png)

## License

MIT
`)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.md", content)
	}
}

func BenchmarkMarkdownLarge(b *testing.B) {
	r := NewDefaultRegistry()
	// Build a large document with repeated sections.
	var sb strings.Builder
	sb.WriteString("# Large Document\n\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("## Section\n\n")
		sb.WriteString("Paragraph with **bold** and *italic* and `code` and [links](https://example.com).\n\n")
		sb.WriteString("| Col A | Col B | Col C |\n|-------|-------|-------|\n")
		sb.WriteString("| one   | two   | three |\n| four  | five  | six   |\n\n")
		sb.WriteString("```go\nfunc example() {\n\treturn nil\n}\n```\n\n")
		sb.WriteString("- item 1\n- item 2\n- item 3\n\n")
	}
	content := []byte(sb.String())
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.md", content)
	}
}

// --- Org-mode benchmarks ---

func BenchmarkOrgSmall(b *testing.B) {
	r := NewDefaultRegistry()
	content := []byte("* Hello\n\nSome text with *bold* and /italic/.\n")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.org", content)
	}
}

func BenchmarkOrgFixture(b *testing.B) {
	r := NewDefaultRegistry()
	content := readBenchdata(b, "README.org")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.org", content)
	}
}

func BenchmarkOrgLarge(b *testing.B) {
	r := NewDefaultRegistry()
	var sb strings.Builder
	sb.WriteString("#+TITLE: Large Document\n\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("* Section\n\n")
		sb.WriteString("Paragraph with *bold* and /italic/ and =code= and [[https://example.com][link]].\n\n")
		sb.WriteString("| Col A | Col B | Col C |\n|-------+-------+-------|\n")
		sb.WriteString("| one   | two   | three |\n| four  | five  | six   |\n\n")
		sb.WriteString("#+begin_src go\nfunc example() {\n\treturn nil\n}\n#+end_src\n\n")
		sb.WriteString("- item 1\n- item 2\n- item 3\n\n")
	}
	content := []byte(sb.String())
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.org", content)
	}
}

// --- External renderer benchmarks ---

func BenchmarkRSTFixture(b *testing.B) {
	r := NewDefaultRegistry()
	if !r.Supported("README.rst") {
		b.Skip("rst2html not installed")
	}
	content := readBenchdata(b, "README.rst")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.rst", content)
	}
}

func BenchmarkAsciiDocFixture(b *testing.B) {
	r := NewDefaultRegistry()
	if !r.Supported("README.adoc") {
		b.Skip("asciidoctor not installed")
	}
	content := readBenchdata(b, "README.asciidoc")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.adoc", content)
	}
}

func BenchmarkPodFixture(b *testing.B) {
	r := NewDefaultRegistry()
	if !r.Supported("README.pod") {
		b.Skip("pod2html not installed")
	}
	content := readBenchdata(b, "README.pod")
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Render("README.pod", content)
	}
}

// --- Body extraction benchmarks ---

func BenchmarkBodyExtractionRST(b *testing.B) {
	r := NewExternalRenderer(ExternalConfig{
		BodyPattern: `(?s)<body>\s*(.*?)\s*</body>`,
	})
	html := `<!DOCTYPE html><html><head><title>Test</title></head><body>
<div class="document">
<h1>Header</h1>
<p>Paragraph with <strong>bold</strong> and <em>italic</em>.</p>
<ul><li>item 1</li><li>item 2</li><li>item 3</li></ul>
</div>
</body></html>`
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		r.extractBody(html)
	}
}

func BenchmarkBodyExtractionPod(b *testing.B) {
	r := NewExternalRenderer(ExternalConfig{
		BodyPattern: `(?s)<body[^>]*>\s*(.*?)\s*</body>`,
	})
	html := `<html><body id="pod"><h1>NAME</h1><p>Hello - a test module</p><h2>DESCRIPTION</h2><p>Some long description here with <code>inline code</code> and other formatting.</p></body></html>`
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		r.extractBody(html)
	}
}
