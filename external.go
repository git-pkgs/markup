package markup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// limitedWriter wraps a writer and returns an error if the limit is exceeded.
type limitedWriter struct {
	w       io.Writer
	n       int
	limit   int
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.n+len(p) > lw.limit {
		return 0, fmt.Errorf("%w: exceeded %d bytes", ErrOutputTooLarge, lw.limit)
	}
	n, err := lw.w.Write(p)
	lw.n += n
	return n, err
}

// InputMode controls how content is passed to an external command.
type InputMode int

const (
	InputStdin    InputMode = iota // Pass content via stdin (default).
	InputTempFile                  // Write to a temp file, replace {file} in args.
)

const defaultTimeout = 30 * time.Second

// maxOutputSize is the maximum allowed output from an external command (2MB).
const maxOutputSize = 2 * 1024 * 1024

// ExternalConfig configures an external renderer.
type ExternalConfig struct {
	Command     string        // Command to execute.
	Args        []string      // Arguments. Use {file} as a placeholder for temp file path.
	InputMode   InputMode     // How to pass content to the command.
	Timeout     time.Duration // Command timeout. Zero means 30s default.
	BodyPattern string        // Regex to extract body from full HTML output. Empty means use full output.
	TempExt     string        // File extension for temp files (e.g. ".pod"). Inferred from command if empty.
}

// ExternalRenderer shells out to a command to render markup.
type ExternalRenderer struct {
	config  ExternalConfig
	bodyRe  *regexp.Regexp
	timeout time.Duration
}

// NewExternalRenderer creates a renderer that delegates to an external command.
func NewExternalRenderer(config ExternalConfig) *ExternalRenderer {
	r := &ExternalRenderer{
		config:  config,
		timeout: config.Timeout,
	}
	if r.timeout == 0 {
		r.timeout = defaultTimeout
	}
	if config.BodyPattern != "" {
		r.bodyRe = regexp.MustCompile(config.BodyPattern)
	}
	return r
}

// Available returns true if the external command is on $PATH.
func (r *ExternalRenderer) Available() bool {
	return toolAvailable(r.config.Command)
}

func (r *ExternalRenderer) Render(content []byte) (string, error) {
	if !r.Available() {
		return "", fmt.Errorf("%w: %s", ErrToolNotFound, r.config.Command)
	}

	var html string
	var err error

	switch r.config.InputMode {
	case InputTempFile:
		html, err = r.renderWithTempFile(content)
	default:
		html, err = r.renderWithStdin(content)
	}

	if err != nil {
		return "", err
	}

	if r.bodyRe != nil {
		html = r.extractBody(html)
	}

	return html, nil
}

func (r *ExternalRenderer) renderWithStdin(content []byte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.config.Command, r.config.Args...)
	cmd.Stdin = bytes.NewReader(content)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdout, limit: maxOutputSize}
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", r.config.Command, err, stderr.String())
	}
	return stdout.String(), nil
}

func (r *ExternalRenderer) renderWithTempFile(content []byte) (string, error) {
	ext := r.config.TempExt
	if ext == "" {
		ext = "." + r.config.Command
	}

	f, err := os.CreateTemp("", "markup-*"+ext)
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(f.Name()) }()

	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return "", err
	}
	_ = f.Close()

	args := make([]string, len(r.config.Args))
	for i, arg := range r.config.Args {
		args[i] = strings.ReplaceAll(arg, "{file}", f.Name())
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.config.Command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdout, limit: maxOutputSize}
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", r.config.Command, err, stderr.String())
	}
	return stdout.String(), nil
}

func (r *ExternalRenderer) extractBody(html string) string {
	// Fast path: try string-based extraction for <body> tags before
	// falling back to regex. Covers the common rst2html/pod2html case.
	if result, ok := extractBodyFast(html); ok {
		return result
	}
	matches := r.bodyRe.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return html
}

// extractBodyFast extracts content between <body> and </body> tags
// using string operations. Returns the content and true if found.
func extractBodyFast(html string) (string, bool) {
	// Find opening <body> or <body ...>
	start := strings.Index(html, "<body")
	if start == -1 {
		return "", false
	}
	// Skip past the closing > of the opening tag
	start = strings.IndexByte(html[start:], '>') + start + 1
	if start <= 0 {
		return "", false
	}

	end := strings.LastIndex(html, "</body>")
	if end == -1 || end <= start {
		return "", false
	}

	body := html[start:end]
	return strings.TrimSpace(body), true
}

// RSTRenderer renders reStructuredText by trying rst2html first,
// then falling back to rst2html.py.
type RSTRenderer struct{}

// NewRSTRenderer creates a renderer for reStructuredText.
func NewRSTRenderer() *RSTRenderer {
	return &RSTRenderer{}
}

func (r *RSTRenderer) Render(content []byte) (string, error) {
	return rstRender(content)
}

func rstRender(content []byte) (string, error) {
	for _, cmd := range []string{"rst2html", "rst2html.py"} {
		if toolAvailable(cmd) {
			r := NewExternalRenderer(ExternalConfig{
				Command:     cmd,
				Args:        []string{"--no-raw", "--no-file-insertion"},
				BodyPattern: `(?s)<body>\s*(.*?)\s*</body>`,
			})
			return r.Render(content)
		}
	}
	return "", fmt.Errorf("%w: rst2html", ErrToolNotFound)
}

// toolCache caches which external tools are available.
var (
	toolCache   sync.Map
)

func toolAvailable(name string) bool {
	if avail, ok := toolCache.Load(name); ok {
		return avail.(bool)
	}
	_, err := exec.LookPath(name)
	avail := err == nil
	toolCache.Store(name, avail)
	return avail
}
