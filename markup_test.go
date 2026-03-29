package markup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading testdata/%s: %v", name, err)
	}
	return data
}

// --- Detect ---

func TestDetect(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
	}{
		{"README.md", FormatMarkdown},
		{"readme.markdown", FormatMarkdown},
		{"CHANGES.mdown", FormatMarkdown},
		{"doc.mkdn", FormatMarkdown},
		{"doc.mdn", FormatMarkdown},
		{"doc.mdtext", FormatMarkdown},
		{"doc.livemd", FormatMarkdown},
		{"README.adoc", FormatAsciiDoc},
		{"README.asciidoc", FormatAsciiDoc},
		{"README.asc", FormatAsciiDoc},
		{"README.rst", FormatRST},
		{"README.rest", FormatRST},
		{"README.rst.txt", FormatRST},
		{"README.textile", FormatTextile},
		{"README.org", FormatOrg},
		{"README.creole", FormatCreole},
		{"README.mediawiki", FormatMediaWiki},
		{"README.wiki", FormatMediaWiki},
		{"README.pod", FormatPod},
		{"README.rdoc", FormatRDoc},
		{"README", FormatUnknown},
		{"README.txt", FormatUnknown},
		{"README.exe", FormatUnknown},
		{"file.go", FormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := Detect(tt.filename)
			if got != tt.want {
				t.Errorf("Detect(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDetectCaseInsensitive(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
	}{
		{"README.MD", FormatMarkdown},
		{"README.Markdown", FormatMarkdown},
		{"README.ADOC", FormatAsciiDoc},
		{"README.RST", FormatRST},
		{"README.RST.TXT", FormatRST},
		{"README.ORG", FormatOrg},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := Detect(tt.filename)
			if got != tt.want {
				t.Errorf("Detect(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// --- Format.String ---

func TestFormatString(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatMarkdown, "Markdown"},
		{FormatOrg, "Org"},
		{FormatAsciiDoc, "AsciiDoc"},
		{FormatRST, "reStructuredText"},
		{FormatTextile, "Textile"},
		{FormatMediaWiki, "MediaWiki"},
		{FormatCreole, "Creole"},
		{FormatPod, "Pod"},
		{FormatRDoc, "RDoc"},
		{FormatUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.format.String()
			if got != tt.want {
				t.Errorf("Format(%d).String() = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

// --- Registry ---

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	_, err := r.Render("README.md", []byte("# hello"))
	if err == nil {
		t.Fatal("expected error from empty registry")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()
	r.Register(FormatMarkdown, NewMarkdownRenderer(), ".md")

	result, err := r.Render("README.md", []byte("**bold**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatMarkdown {
		t.Errorf("Format = %v, want FormatMarkdown", result.Format)
	}
	if !strings.Contains(result.HTML, "<strong>bold</strong>") {
		t.Errorf("expected bold in HTML, got: %s", result.HTML)
	}
}

func TestRegistryRegisterFilename(t *testing.T) {
	r := NewRegistry()
	r.RegisterFilename(FormatMarkdown, NewMarkdownRenderer(), "README")

	result, err := r.Render("README", []byte("**bold**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatMarkdown {
		t.Errorf("Format = %v, want FormatMarkdown", result.Format)
	}
}

func TestRegistryFilenameOverExtension(t *testing.T) {
	r := NewRegistry()
	called := false
	r.RegisterFilename(FormatMarkdown, RendererFunc(func(content []byte) (string, error) {
		called = true
		return "<p>filename match</p>", nil
	}), "readme.md")
	r.Register(FormatMarkdown, NewMarkdownRenderer(), ".md")

	_, err := r.Render("README.md", []byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected filename match to take priority over extension match")
	}
}

func TestRegistryUnsupportedFormat(t *testing.T) {
	r := NewDefaultRegistry()
	_, err := r.Render("README.xyz", []byte("hello"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected ErrUnsupportedFormat, got: %v", err)
	}
}

func TestRegistrySupported(t *testing.T) {
	r := NewDefaultRegistry()

	if !r.Supported("README.md") {
		t.Error("expected Markdown to be supported")
	}
	if !r.Supported("README.org") {
		t.Error("expected Org to be supported")
	}
	if r.Supported("README.xyz") {
		t.Error("expected .xyz to not be supported")
	}
}

// --- Markdown rendering ---

func TestRenderMarkdown(t *testing.T) {
	r := NewDefaultRegistry()
	content := readTestdata(t, "README.markdown")
	result, err := r.Render("README.markdown", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatMarkdown {
		t.Errorf("Format = %v, want FormatMarkdown", result.Format)
	}
	if !strings.Contains(result.HTML, "<li>One</li>") {
		t.Errorf("expected list items in HTML, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<li>Two</li>") {
		t.Errorf("expected list items in HTML, got: %s", result.HTML)
	}
}

func TestRenderMarkdownHeadings(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("# Hello\n\n## World\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<h1>Hello</h1>") {
		t.Errorf("expected h1 tag, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<h2>World</h2>") {
		t.Errorf("expected h2 tag, got: %s", result.HTML)
	}
}

func TestRenderMarkdownInlineFormatting(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("This is **bold** and *italic* and `code`.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<strong>bold</strong>") {
		t.Errorf("expected strong tag, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<em>italic</em>") {
		t.Errorf("expected em tag, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<code>code</code>") {
		t.Errorf("expected code tag, got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMTable(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("| a | b |\n|---|---|\n| 1 | 2 |\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<table>") {
		t.Errorf("expected table tag, got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMTaskList(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("- [x] Done\n- [ ] Todo\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "checkbox") {
		t.Errorf("expected checkbox in HTML, got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMStrikethrough(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("~~deleted~~\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<del>deleted</del>") {
		t.Errorf("expected del tag, got: %s", result.HTML)
	}
}

func TestRenderMarkdownGFMAutolink(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("Visit https://example.com for more.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, `href="https://example.com"`) {
		t.Errorf("expected autolink, got: %s", result.HTML)
	}
}

func TestRenderMarkdownUnsafeHTML(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte(`<div class="custom">Hello</div>`+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, `<div class="custom">Hello</div>`) {
		t.Errorf("expected raw HTML passthrough, got: %s", result.HTML)
	}
}

func TestRenderMarkdownCodeBlock(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("```go\nfunc main() {}\n```\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<code") {
		t.Errorf("expected code block, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "func main()") {
		t.Errorf("expected code content, got: %s", result.HTML)
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if result.HTML != "" {
		t.Errorf("expected empty HTML for empty content, got: %q", result.HTML)
	}
}

func TestRenderMarkdownLinks(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("[example](https://example.com)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, `<a href="https://example.com">example</a>`) {
		t.Errorf("expected link, got: %s", result.HTML)
	}
}

func TestRenderMarkdownImages(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.md", []byte("![alt](image.png)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, `<img src="image.png" alt="alt"`) {
		t.Errorf("expected image tag, got: %s", result.HTML)
	}
}

// --- Org-mode rendering ---

func TestRenderOrg(t *testing.T) {
	r := NewDefaultRegistry()
	content := readTestdata(t, "README.org")
	result, err := r.Render("README.org", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatOrg {
		t.Errorf("Format = %v, want FormatOrg", result.Format)
	}
	if !strings.Contains(result.HTML, "Description") {
		t.Errorf("expected heading in HTML, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "org-ruby") {
		t.Errorf("expected title in HTML, got: %s", result.HTML)
	}
}

func TestRenderOrgHeadings(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.org", []byte("* Heading 1\n** Heading 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "Heading 1") {
		t.Errorf("expected h1, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "Heading 2") {
		t.Errorf("expected h2, got: %s", result.HTML)
	}
}

func TestRenderOrgInlineFormatting(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.org", []byte("This is *bold* and /italic/ and =code=.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<strong>bold</strong>") {
		t.Errorf("expected bold, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "<em>italic</em>") {
		t.Errorf("expected italic, got: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "code") || !strings.Contains(result.HTML, "verbatim") {
		t.Errorf("expected verbatim code, got: %s", result.HTML)
	}
}

func TestRenderOrgCodeBlock(t *testing.T) {
	r := NewDefaultRegistry()
	content := []byte("#+begin_src ruby\nputs \"hello\"\n#+end_src\n")
	result, err := r.Render("README.org", content)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "puts") {
		t.Errorf("expected code content, got: %s", result.HTML)
	}
}

func TestRenderOrgTable(t *testing.T) {
	r := NewDefaultRegistry()
	content := []byte("| Name | Value |\n|------+-------|\n| foo  | bar   |\n")
	result, err := r.Render("README.org", content)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<table>") {
		t.Errorf("expected table, got: %s", result.HTML)
	}
}

func TestRenderOrgLinks(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.org", []byte("[[https://example.com][example]]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "https://example.com") {
		t.Errorf("expected link, got: %s", result.HTML)
	}
}

func TestRenderOrgLists(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.org", []byte("- item 1\n- item 2\n- item 3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<li>") {
		t.Errorf("expected list items, got: %s", result.HTML)
	}
}

func TestRenderOrgEmpty(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("README.org", []byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.HTML) != "" {
		t.Errorf("expected empty HTML for empty content, got: %q", result.HTML)
	}
}

// --- External renderers ---

func TestRenderAsciiDoc(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.adoc") {
		t.Skip("asciidoctor not installed")
	}
	content := readTestdata(t, "README.asciidoc")
	result, err := r.Render("README.adoc", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatAsciiDoc {
		t.Errorf("Format = %v, want FormatAsciiDoc", result.Format)
	}
	if !strings.Contains(result.HTML, "First Section") {
		t.Errorf("expected heading in HTML, got: %s", result.HTML)
	}
}

func TestRenderRST(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.rst") {
		t.Skip("rst2html not installed")
	}
	content := readTestdata(t, "README.rst")
	result, err := r.Render("README.rst", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatRST {
		t.Errorf("Format = %v, want FormatRST", result.Format)
	}
	if !strings.Contains(result.HTML, "Header 1") {
		t.Errorf("expected heading in HTML, got: %s", result.HTML)
	}
	// Body extraction should strip the <html>/<body> wrapper.
	if strings.Contains(result.HTML, "<!DOCTYPE") || strings.Contains(result.HTML, "<html") {
		t.Errorf("expected body-only HTML, got full document: %s", result.HTML[:200])
	}
}

func TestRenderRSTTxt(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.rst") {
		t.Skip("rst2html not installed")
	}
	content := readTestdata(t, "README.rst.txt")
	result, err := r.Render("README.rst.txt", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatRST {
		t.Errorf("Format = %v, want FormatRST", result.Format)
	}
}

func TestRenderPod(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.pod") {
		t.Skip("pod2html not installed")
	}
	content := readTestdata(t, "README.pod")
	result, err := r.Render("README.pod", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatPod {
		t.Errorf("Format = %v, want FormatPod", result.Format)
	}
	if !strings.Contains(result.HTML, "Matrixy") {
		t.Errorf("expected heading in HTML, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, "<html") {
		t.Errorf("expected body-only HTML, got full document")
	}
}

func TestRenderTextile(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.textile") {
		t.Skip("textile renderer not installed")
	}
	content := readTestdata(t, "README.textile")
	result, err := r.Render("README.textile", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatTextile {
		t.Errorf("Format = %v, want FormatTextile", result.Format)
	}
}

func TestRenderMediaWiki(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.mediawiki") {
		t.Skip("mediawiki renderer not installed")
	}
	content := readTestdata(t, "README.mediawiki")
	result, err := r.Render("README.mediawiki", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatMediaWiki {
		t.Errorf("Format = %v, want FormatMediaWiki", result.Format)
	}
}

func TestRenderCreole(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.creole") {
		t.Skip("creole renderer not installed")
	}
	content := readTestdata(t, "README.creole")
	result, err := r.Render("README.creole", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatCreole {
		t.Errorf("Format = %v, want FormatCreole", result.Format)
	}
}

func TestRenderRDoc(t *testing.T) {
	r := NewDefaultRegistry()
	if !r.Supported("README.rdoc") {
		t.Skip("rdoc renderer not installed")
	}
	content := readTestdata(t, "README.rdoc")
	result, err := r.Render("README.rdoc", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatRDoc {
		t.Errorf("Format = %v, want FormatRDoc", result.Format)
	}
}

// --- Input size limit ---

func TestRenderInputTooLarge(t *testing.T) {
	r := NewRegistry()
	r.Register(FormatMarkdown, NewMarkdownRenderer(), ".md")
	r.SetMaxInputSize(100)

	_, err := r.Render("README.md", make([]byte, 101))
	if err == nil {
		t.Fatal("expected error for oversized input")
	}
	if !strings.Contains(err.Error(), "input exceeds") {
		t.Errorf("expected input too large error, got: %v", err)
	}
}

func TestRenderInputWithinLimit(t *testing.T) {
	r := NewRegistry()
	r.Register(FormatMarkdown, NewMarkdownRenderer(), ".md")
	r.SetMaxInputSize(100)

	_, err := r.Render("README.md", []byte("# Hello\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderInputLimitDisabled(t *testing.T) {
	r := NewRegistry()
	r.Register(FormatMarkdown, NewMarkdownRenderer(), ".md")
	r.SetMaxInputSize(0)

	// Should not error even with large input when limit is disabled.
	_, err := r.Render("README.md", make([]byte, 1024*1024))
	if err != nil {
		t.Fatalf("unexpected error with disabled limit: %v", err)
	}
}

func TestDefaultMaxInputSize(t *testing.T) {
	r := NewRegistry()
	if r.maxInputSize != DefaultMaxInputSize {
		t.Errorf("default max input size = %d, want %d", r.maxInputSize, DefaultMaxInputSize)
	}
}

// --- Error handling ---

func TestRenderToolNotFound(t *testing.T) {
	r := NewRegistry()
	r.Register(FormatAsciiDoc, NewExternalRenderer(ExternalConfig{
		Command: "nonexistent-tool-that-does-not-exist",
	}), ".adoc")

	_, err := r.Render("README.adoc", []byte("= Hello"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected tool not found error, got: %v", err)
	}
}

func TestRenderUnsupportedFormat(t *testing.T) {
	r := NewDefaultRegistry()
	_, err := r.Render("README.xyz", []byte("hello"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

// --- External renderer ---

func TestExternalRendererAvailable(t *testing.T) {
	r := NewExternalRenderer(ExternalConfig{
		Command: "sh",
	})
	if !r.Available() {
		t.Error("expected sh to be available")
	}

	r2 := NewExternalRenderer(ExternalConfig{
		Command: "nonexistent-tool-12345",
	})
	if r2.Available() {
		t.Error("expected nonexistent tool to be unavailable")
	}
}

func TestExternalRendererBodyExtraction(t *testing.T) {
	r := NewExternalRenderer(ExternalConfig{
		Command:     "echo",
		Args:        []string{"<html><body>\n<h1>Hello</h1>\n</body></html>"},
		BodyPattern: `(?s)<body[^>]*>\s*(.*?)\s*</body>`,
	})

	html, err := r.Render([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "<h1>Hello</h1>") {
		t.Errorf("expected extracted body, got: %s", html)
	}
	if strings.Contains(html, "<html>") {
		t.Error("expected html wrapper to be stripped")
	}
}

// --- RendererFunc ---

func TestRendererFunc(t *testing.T) {
	fn := RendererFunc(func(content []byte) (string, error) {
		return "<p>" + string(content) + "</p>", nil
	})

	r := NewRegistry()
	r.Register(FormatMarkdown, fn, ".custom")

	result, err := r.Render("test.custom", []byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HTML != "<p>hello</p>" {
		t.Errorf("expected <p>hello</p>, got: %s", result.HTML)
	}
}

// --- Multiple registries ---

func TestMultipleRegistries(t *testing.T) {
	r1 := NewRegistry()
	r1.Register(FormatMarkdown, RendererFunc(func(content []byte) (string, error) {
		return "<p>registry 1</p>", nil
	}), ".md")

	r2 := NewRegistry()
	r2.Register(FormatMarkdown, RendererFunc(func(content []byte) (string, error) {
		return "<p>registry 2</p>", nil
	}), ".md")

	res1, _ := r1.Render("README.md", []byte("hello"))
	res2, _ := r2.Render("README.md", []byte("hello"))

	if res1.HTML == res2.HTML {
		t.Error("expected different registries to produce different output")
	}
}

// --- Path handling ---

func TestRenderWithPath(t *testing.T) {
	r := NewDefaultRegistry()
	result, err := r.Render("docs/api/README.md", []byte("# API\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.HTML, "<h1>API</h1>") {
		t.Errorf("expected rendering with path prefix, got: %s", result.HTML)
	}
}
