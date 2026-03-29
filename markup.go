// Package markup renders markup files to HTML.
//
// Given a filename and its contents, it picks the right renderer and
// produces HTML. Markdown and Org-mode are rendered natively in Go.
// Other formats shell out to external tools (asciidoctor, rst2html,
// pod2html, etc).
//
// The library has no global state. Construct a Registry, configure it,
// and call Render. Post-processing (sanitization, link rewriting,
// mention linking) is the consumer's responsibility.
package markup

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Errors returned by Render.
var (
	ErrUnsupportedFormat = errors.New("unsupported markup format")
	ErrToolNotFound      = errors.New("external tool not found")
	ErrInputTooLarge     = errors.New("input exceeds maximum size")
	ErrOutputTooLarge    = errors.New("output exceeds maximum size")
)

// DefaultMaxInputSize is the default maximum input size (1MB).
const DefaultMaxInputSize = 1024 * 1024

// Format represents a markup format.
type Format int

const (
	FormatUnknown   Format = iota
	FormatMarkdown         // .md, .markdown, .mdown, .mkdn, .mdn, .mdtext, .livemd
	FormatOrg              // .org
	FormatAsciiDoc         // .adoc, .asciidoc, .asc
	FormatRST              // .rst, .rest, .rst.txt
	FormatTextile          // .textile
	FormatMediaWiki        // .mediawiki, .wiki
	FormatCreole           // .creole
	FormatPod              // .pod
	FormatRDoc             // .rdoc
)

// String returns the human-readable name for the format.
func (f Format) String() string {
	switch f {
	case FormatMarkdown:
		return "Markdown"
	case FormatOrg:
		return "Org"
	case FormatAsciiDoc:
		return "AsciiDoc"
	case FormatRST:
		return "reStructuredText"
	case FormatTextile:
		return "Textile"
	case FormatMediaWiki:
		return "MediaWiki"
	case FormatCreole:
		return "Creole"
	case FormatPod:
		return "Pod"
	case FormatRDoc:
		return "RDoc"
	default:
		return ""
	}
}

// Result holds the rendered output.
type Result struct {
	HTML   string
	Format Format
}

// Renderer converts markup content to HTML.
type Renderer interface {
	Render(content []byte) (string, error)
}

// RendererFunc adapts a function to the Renderer interface.
type RendererFunc func(content []byte) (string, error)

func (f RendererFunc) Render(content []byte) (string, error) {
	return f(content)
}

// Registry maps file extensions and filenames to renderers.
type Registry struct {
	extensions   map[string]registryEntry
	filenames    map[string]registryEntry
	maxInputSize int
}

type registryEntry struct {
	format   Format
	renderer Renderer
}

// NewRegistry creates an empty registry with the default max input size (25MB).
func NewRegistry() *Registry {
	return &Registry{
		extensions:   make(map[string]registryEntry),
		filenames:    make(map[string]registryEntry),
		maxInputSize: DefaultMaxInputSize,
	}
}

// SetMaxInputSize sets the maximum allowed input size in bytes.
// Zero disables the limit.
func (r *Registry) SetMaxInputSize(n int) {
	r.maxInputSize = n
}

// NewDefaultRegistry creates a registry with all supported renderers.
// Native renderers (Markdown, Org-mode) are always available.
// External renderers (AsciiDoc, reStructuredText, Pod) are registered
// but return ErrToolNotFound if the required tool is not on $PATH.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()

	r.Register(FormatMarkdown, NewMarkdownRenderer(),
		".md", ".markdown", ".mdown", ".mkdn", ".mdn", ".mdtext", ".livemd")

	r.Register(FormatOrg, NewOrgRenderer(),
		".org")

	r.Register(FormatAsciiDoc, NewExternalRenderer(ExternalConfig{
		Command: "asciidoctor",
		Args:    []string{"-s", "-o", "-", "-"},
	}), ".adoc", ".asciidoc", ".asc")

	r.Register(FormatRST, NewRSTRenderer(),
		".rst", ".rest")

	r.Register(FormatPod, NewExternalRenderer(ExternalConfig{
		Command:   "pod2html",
		InputMode: InputTempFile,
		Args:      []string{"--infile={file}", "--quiet"},
		BodyPattern: `(?s)<body[^>]*>\s*(.*?)\s*</body>`,
	}), ".pod")

	r.Register(FormatTextile, NewExternalRenderer(ExternalConfig{
		Command: "textile",
	}), ".textile")

	r.Register(FormatMediaWiki, NewExternalRenderer(ExternalConfig{
		Command: "mediawiki",
	}), ".mediawiki", ".wiki")

	r.Register(FormatCreole, NewExternalRenderer(ExternalConfig{
		Command: "creole",
	}), ".creole")

	r.Register(FormatRDoc, NewExternalRenderer(ExternalConfig{
		Command: "rdoc",
	}), ".rdoc")

	return r
}

// Register adds a renderer for the given format and file extensions.
// Extensions should include the leading dot (e.g. ".md").
func (r *Registry) Register(format Format, renderer Renderer, extensions ...string) {
	for _, ext := range extensions {
		r.extensions[strings.ToLower(ext)] = registryEntry{format: format, renderer: renderer}
	}
}

// RegisterFilename adds a renderer for an exact filename match.
func (r *Registry) RegisterFilename(format Format, renderer Renderer, filenames ...string) {
	for _, name := range filenames {
		r.filenames[strings.ToLower(name)] = registryEntry{format: format, renderer: renderer}
	}
}

// Render converts markup content to HTML based on the filename.
// Returns ErrUnsupportedFormat if no renderer matches.
// Returns ErrToolNotFound if the format needs an external tool that isn't installed.
func (r *Registry) Render(filename string, content []byte) (Result, error) {
	if r.maxInputSize > 0 && len(content) > r.maxInputSize {
		return Result{}, fmt.Errorf("%w: %d bytes (max %d)", ErrInputTooLarge, len(content), r.maxInputSize)
	}

	entry, ok := r.lookup(filename)
	if !ok {
		return Result{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, filename)
	}

	html, err := entry.renderer.Render(content)
	if err != nil {
		return Result{}, err
	}

	return Result{
		HTML:   html,
		Format: entry.format,
	}, nil
}

// Supported returns true if the filename can be rendered by this registry,
// including checking that any required external tools are installed.
func (r *Registry) Supported(filename string) bool {
	entry, ok := r.lookup(filename)
	if !ok {
		return false
	}
	if ext, ok := entry.renderer.(*ExternalRenderer); ok {
		return ext.Available()
	}
	return true
}

func (r *Registry) lookup(filename string) (registryEntry, bool) {
	lower := strings.ToLower(filepath.Base(filename))

	// Exact filename match first.
	if entry, ok := r.filenames[lower]; ok {
		return entry, true
	}

	// Special case: .rst.txt
	if strings.HasSuffix(lower, ".rst.txt") {
		if entry, ok := r.extensions[".rst"]; ok {
			return entry, true
		}
	}

	ext := filepath.Ext(lower)
	if entry, ok := r.extensions[ext]; ok {
		return entry, true
	}

	return registryEntry{}, false
}

// Detect identifies the markup format from a filename using the default
// extension mapping. This is a convenience for callers who only need
// format detection without a full registry.
func Detect(filename string) Format {
	lower := strings.ToLower(filename)

	if strings.HasSuffix(lower, ".rst.txt") {
		return FormatRST
	}

	ext := filepath.Ext(lower)
	if f, ok := defaultExtensions[ext]; ok {
		return f
	}
	return FormatUnknown
}

var defaultExtensions = map[string]Format{
	".md":        FormatMarkdown,
	".markdown":  FormatMarkdown,
	".mdown":     FormatMarkdown,
	".mkdn":      FormatMarkdown,
	".mdn":       FormatMarkdown,
	".mdtext":    FormatMarkdown,
	".livemd":    FormatMarkdown,
	".org":       FormatOrg,
	".adoc":      FormatAsciiDoc,
	".asciidoc":  FormatAsciiDoc,
	".asc":       FormatAsciiDoc,
	".rst":       FormatRST,
	".rest":      FormatRST,
	".textile":   FormatTextile,
	".mediawiki": FormatMediaWiki,
	".wiki":      FormatMediaWiki,
	".creole":    FormatCreole,
	".pod":       FormatPod,
	".rdoc":      FormatRDoc,
}
