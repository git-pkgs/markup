# markup

A Go library for rendering markup files to HTML. Given a filename and its contents, it picks the right renderer and produces HTML.

Markdown and Org-mode are rendered natively in Go. Other formats (AsciiDoc, reStructuredText, Pod) shell out to external tools. The library has no global state, no forge-specific logic, and no opinion on post-processing.

## Installation

```sh
go get github.com/git-pkgs/markup
```

## Usage

```go
r := markup.NewDefaultRegistry()

result, err := r.Render("README.md", content)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.HTML)
fmt.Println(result.Format) // "Markdown"
```

The default registry comes with all supported formats pre-configured. Native formats (Markdown, Org-mode) always work. External formats return `markup.ErrToolNotFound` if the required tool isn't on `$PATH`.

## Supported formats

| Format | Extensions | Renderer |
|---|---|---|
| Markdown | .md, .markdown, .mdown, .mkdn, .mdn, .mdtext, .livemd | goldmark (native) |
| Org-mode | .org | go-org (native) |
| AsciiDoc | .adoc, .asciidoc, .asc | asciidoctor (external) |
| reStructuredText | .rst, .rest, .rst.txt | rst2html (external) |
| Pod | .pod | pod2html (external) |
| Textile | .textile | consumer-provided (external) |
| MediaWiki | .mediawiki, .wiki | consumer-provided (external) |
| Creole | .creole | consumer-provided (external) |
| RDoc | .rdoc | consumer-provided (external) |

## Format detection

```go
format := markup.Detect("README.org") // markup.FormatOrg
format.String()                       // "Org"
```

## Custom renderers

Register your own renderer for any format:

```go
r := markup.NewRegistry()

r.Register(markup.FormatMarkdown, markup.NewMarkdownRenderer(), ".md", ".markdown")

r.Register(markup.FormatTextile, markup.NewExternalRenderer(markup.ExternalConfig{
    Command: "pandoc",
    Args:    []string{"-f", "textile", "-t", "html"},
}), ".textile")

// Exact filename matching
r.RegisterFilename(markup.FormatMarkdown, markup.NewMarkdownRenderer(), "README")
```

## Errors

```go
result, err := r.Render("README.adoc", content)
if errors.Is(err, markup.ErrUnsupportedFormat) {
    // no renderer registered for this extension
}
if errors.Is(err, markup.ErrToolNotFound) {
    // renderer registered but external tool not on $PATH
}
```

## Installing external tools

The native formats (Markdown, Org-mode) need no external tools. For the others:

| Tool | Install | Formats |
|---|---|---|
| asciidoctor | `gem install asciidoctor` | AsciiDoc |
| rst2html | `pip install docutils` or `apt install python3-docutils` | reStructuredText |
| pod2html | Included with Perl | Pod |

The included `Dockerfile` builds an image with all optional tools installed.

## Benchmarks

```
BenchmarkMarkdownSmall     387460    3155 ns/op     7448 B/op     33 allocs/op
BenchmarkMarkdownMedium     43614   27558 ns/op    30626 B/op    196 allocs/op
BenchmarkMarkdownLarge        813 1466013 ns/op  1561079 B/op  11432 allocs/op
BenchmarkOrgSmall          104258   11548 ns/op    11897 B/op    131 allocs/op
BenchmarkOrgFixture          1729  688588 ns/op   236167 B/op   2084 allocs/op
```

External renderers are dominated by subprocess overhead (~30ms for pod2html, ~90ms for rst2html, ~120ms for asciidoctor).

## License

[MIT](LICENSE).
