// implements the go-template output format, similar to kubectl's

package util

import (
	"bytes"
	"text/template"
)

type goTemplateOutput struct {
	RawTemplate string
	template    *template.Template
}

func (g goTemplateOutput) Marshal(r DNSResult) string {
	var tpl bytes.Buffer
	err := g.template.Execute(&tpl, r)
	if err != nil {
		return ""
	}
	return tpl.String()
}

func (g *goTemplateOutput) Init() (string, error) {
	var err error
	g.template, err = template.New("gotemplate").Parse(g.RawTemplate) // todo:Fix
	if err != nil {
		return "", err
	}
	return "", nil
}
