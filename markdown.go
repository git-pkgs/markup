package markup

import (
	"bytes"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// MarkdownRenderer renders Markdown to HTML using goldmark.
type MarkdownRenderer struct {
	md goldmark.Markdown
}

// NewMarkdownRenderer creates a Markdown renderer with GFM extensions
// (tables, strikethrough, task lists, autolinks) and unsafe HTML passthrough.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(
				html.WithUnsafe(),
			),
		),
	}
}

func (r *MarkdownRenderer) Render(content []byte) (string, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := r.md.Convert(content, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
