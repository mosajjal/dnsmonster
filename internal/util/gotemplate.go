/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

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

func (g goTemplateOutput) Marshal(r DNSResult) []byte {
	var tpl bytes.Buffer
	err := g.template.Execute(&tpl, r)
	if err != nil {
		return nil
	}
	return tpl.Bytes()
}

func (g *goTemplateOutput) Init() (string, error) {
	var err error
	g.template, err = template.New("gotemplate").Parse(g.RawTemplate) // todo:Fix
	if err != nil {
		return "", err
	}
	return "", nil
}
// vim: foldmethod=marker
