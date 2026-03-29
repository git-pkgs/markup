package markup

import (
	"bytes"

	"github.com/niklasfasching/go-org/org"
)

// OrgRenderer renders Org-mode files to HTML using go-org.
type OrgRenderer struct{}

// NewOrgRenderer creates an Org-mode renderer.
func NewOrgRenderer() *OrgRenderer {
	return &OrgRenderer{}
}

func (r *OrgRenderer) Render(content []byte) (string, error) {
	doc := org.New().Parse(bytes.NewReader(content), "")
	html, err := doc.Write(org.NewHTMLWriter())
	if err != nil {
		return "", err
	}
	return html, nil
}
