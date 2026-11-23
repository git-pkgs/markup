package markup

import (
	"bytes"
	"errors"

	"github.com/niklasfasching/go-org/org"
)

var errFileAccessDisabled = errors.New("file access disabled")

// OrgRenderer renders Org-mode files to HTML using go-org.
type OrgRenderer struct{}

// NewOrgRenderer creates an Org-mode renderer.
func NewOrgRenderer() *OrgRenderer {
	return &OrgRenderer{}
}

func (r *OrgRenderer) Render(content []byte) (string, error) {
	conf := org.New().Silent()
	// #+INCLUDE and #+SETUPFILE would otherwise read arbitrary local files.
	conf.ReadFile = func(string) ([]byte, error) {
		return nil, errFileAccessDisabled
	}
	doc := conf.Parse(bytes.NewReader(content), "")
	html, err := doc.Write(org.NewHTMLWriter())
	if err != nil {
		return "", err
	}
	return html, nil
}
