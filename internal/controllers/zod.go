package controllers

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"zodGeneratorCLI/internal/models"
)

type scheme struct {
	Imports []models.TsImport
	Raw     string
}

type Zod interface {
	Convert(ctx context.Context, files []models.ParsedFile) ([]models.ZodFile, error)
}

type zod struct {
	schemes map[string]scheme
}

func NewZod() Zod {
	schemes := map[string]scheme{
		"number":    {Raw: "z.number()"},
		"string":    {Raw: "z.string()"},
		"boolean":   {Raw: "z.boolean()"},
		"any":       {Raw: "z.any()"},
		"null":      {Raw: "z.null()"},
		"undefined": {Raw: "z.undefined()"},
		"DateTime": {
			Raw: "z.string().datetime().transform(dt => DateTime.fromISO(dt))",
			Imports: []models.TsImport{
				{
					Import: []string{"DateTime"},
					From:   "luxon",
				},
			}},
	}

	return &zod{
		schemes: schemes,
	}
}

func (z *zod) Convert(_ context.Context, files []models.ParsedFile) ([]models.ZodFile, error) {
	parsedModels := z.flatten(files)
	parsedModels, err := z.sortModels(parsedModels)
	if err != nil {
		return nil, err
	}

	zodModels := make([]models.ZodModel, len(parsedModels))
	for i, model := range parsedModels {
		zodModel, err := z.convertModel(model)
		if err != nil {
			return nil, err
		}

		zodModels[i] = zodModel
	}

	return z.combineModels(zodModels)
}

func (z *zod) flatten(files []models.ParsedFile) []models.ParsedModel {
	var parsedModels []models.ParsedModel
	for _, file := range files {
		parsedModels = append(parsedModels, file.Models...)
	}

	return parsedModels
}

func (z *zod) sortModels(parsedModels []models.ParsedModel) ([]models.ParsedModel, error) {
	visited := make(map[string]bool)
	result := make([]models.ParsedModel, 0)

	var visit func(element models.ParsedModel) error

	visit = func(element models.ParsedModel) error {
		if visited[element.Name] {
			return nil
		}

		for _, dependency := range element.DependsOn {
			i := slices.IndexFunc(parsedModels, func(model models.ParsedModel) bool {
				return model.Name == dependency
			})
			if i == -1 {
				return errors.New("not found interface for " + dependency)
			}

			depElement := parsedModels[i]
			if err := visit(depElement); err != nil {
				return err
			}
		}

		visited[element.Name] = true
		result = append(result, element)
		return nil
	}

	for _, element := range parsedModels {
		if err := visit(element); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (z *zod) convertModel(model models.ParsedModel) (models.ZodModel, error) {
	fields, imports, err := z.convertFields(model.ParsedFields)
	if err != nil {
		return models.ZodModel{}, err
	}

	name := model.Name + "Scheme"

	builder := strings.Builder{}
	builder.WriteString("export const ")
	builder.WriteString(name)
	builder.WriteString(" = z.object({\n")

	for _, field := range fields {
		builder.WriteString("\t")
		builder.WriteString(field.Name)
		builder.WriteString(": ")
		builder.WriteString(field.Raw)
		builder.WriteString(",\n")
	}

	builder.WriteString("});")

	z.schemes[model.Name] = scheme{
		Imports: []models.TsImport{
			{
				Import:        []string{name},
				From:          z.getImportPath(model.Path),
				ParseFilepath: true,
			},
		},
		Raw: name,
	}

	return models.ZodModel{
		Path:    model.Path,
		Imports: imports,
		Name:    name,
		Fields:  fields,
		Raw:     builder.String(),
	}, nil
}

func (z *zod) convertFields(fields []models.ParsedField) ([]models.ZodField, []models.TsImport, error) {
	var imports []models.TsImport
	zodFields := make([]models.ZodField, len(fields))
	for i, field := range fields {
		zodField, err := z.convertField(field)
		if err != nil {
			return nil, nil, err
		}

		imports = append(imports, zodField.Imports...)
		zodFields[i] = zodField
	}

	return zodFields, imports, nil
}

func (z *zod) convertField(field models.ParsedField) (models.ZodField, error) {
	var imports []models.TsImport

	builder := strings.Builder{}
	if field.IsArray {
		builder.WriteString("z.array(")
	}

	{
		s, err := z.parseType(field.Types[0])
		if err != nil {
			return models.ZodField{}, err
		}

		imports = append(imports, s.Imports...)
		builder.WriteString(s.Raw)
	}

	for _, type_ := range field.Types[1:] {
		s, err := z.parseType(type_)
		if err != nil {
			return models.ZodField{}, err
		}

		imports = append(imports, s.Imports...)

		builder.WriteString(".or(")
		builder.WriteString(s.Raw)
		builder.WriteString(")")
	}

	if field.IsArray {
		builder.WriteString(")")
	}

	if field.Nullable {
		builder.WriteString(".nullable()")
	}

	return models.ZodField{
		Imports: imports,
		Name:    field.Name,
		Raw:     builder.String(),
	}, nil
}

func (z *zod) parseType(type_ string) (scheme, error) {
	s, ok := z.schemes[type_]
	if !ok {
		return scheme{}, errors.New("cannot find scheme for type: " + type_)
	}

	return s, nil
}

func (z *zod) combineModels(model []models.ZodModel) ([]models.ZodFile, error) {
	dict := make(map[string]models.ZodFile)
	for _, m := range model {
		file, ok := dict[m.Path]
		if !ok {
			file = models.ZodFile{
				Path: z.getFilename(m.Path),
			}
		}

		for _, tsImport := range m.Imports {
			if tsImport.From+".ts" == file.Path || slices.ContainsFunc(file.Imports, func(i models.TsImport) bool {
				return i.Import[0] == tsImport.Import[0]
			}) {
				continue
			}
			file.Imports = append(file.Imports, tsImport)
		}

		file.Models = append(file.Models, m)

		dict[m.Path] = file
	}

	files := make([]models.ZodFile, 0, len(dict))
	for _, file := range dict {
		files = append(files, file)
	}

	return files, nil
}

func (z *zod) getFilename(file string) string {
	return z.getImportPath(file) + filepath.Ext(file)
}

func (z *zod) getImportPath(file string) string {
	filename := filepath.Base(file)
	ext := filepath.Ext(filename)

	return filepath.Dir(file) + "/schemes/" + filename[:len(filename)-len(ext)] + ".scheme"
}
