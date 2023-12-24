package models

import (
	"path/filepath"
	"strings"
)

type TsImport struct {
	Import        []string
	From          string
	ParseFilepath bool
}

func (i *TsImport) Raw(path string) string {
	p := i.From
	if i.ParseFilepath {
		p, _ = filepath.Rel(path, i.From)
		if strings.HasPrefix(p, "../") {
			p = p[1:]
		}
	}

	builder := strings.Builder{}

	builder.WriteString("import { ")
	builder.WriteString(strings.Join(i.Import, ", "))
	builder.WriteString(" } from '")
	builder.WriteString(p)
	builder.WriteString("';")

	return builder.String()
}

type ZodModel struct {
	Path    string
	Imports []TsImport
	Name    string
	Fields  []ZodField
	Raw     string
}

type ZodField struct {
	Imports []TsImport
	Name    string
	Raw     string
}

type ZodFile struct {
	Models  []ZodModel
	Path    string
	Imports []TsImport
}
