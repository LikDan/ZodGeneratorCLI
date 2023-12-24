package controllers

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"zodGeneratorCLI/internal/models"
)

type Parser interface {
	ParseFiles(ctx context.Context, filenames []string) ([]models.ParsedFile, error)
}

type parser struct {
	InterfaceRegex   *regexp.Regexp
	InstructionRegex *regexp.Regexp
}

func NewParser() Parser {
	return &parser{
		InterfaceRegex:   regexp.MustCompile("interface\\s+(\\w+)\\s*\\{\\s*([^{}]+(\\{\\s*[^{}]+}\\s*)?[^{}]+)*}\\s*"),
		InstructionRegex: regexp.MustCompile("(;\\n|[;\\n])"),
	}
}

func (p *parser) ParseFiles(_ context.Context, filenames []string) ([]models.ParsedFile, error) {
	var files []models.ParsedFile
	for _, file := range filenames {
		parsedFile, err := p.parseFile(file)
		if err != nil {
			return nil, err
		}

		files = append(files, parsedFile)
	}

	return files, nil
}

func (p *parser) parseFile(path string) (models.ParsedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.ParsedFile{}, err
	}

	matches := p.InterfaceRegex.FindAll(data, -1)
	parsedModels := make([]models.ParsedModel, len(matches))
	for i, match := range matches {
		model, err := p.parseInterface(match)
		if err != nil {
			return models.ParsedFile{}, err
		}

		model.Path = path
		parsedModels[i] = model
	}

	return models.ParsedFile{
		Path:   path,
		Models: parsedModels,
	}, nil
}

func (p *parser) parseInterface(match []byte) (models.ParsedModel, error) {
	lines := p.splitByInstructions(match)
	if len(lines) < 2 {
		return models.ParsedModel{}, errors.New("lines len less than 2")
	}
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(lines[0], "interface"), "{"))
	fields, depends, err := p.parseFields(lines[1 : len(lines)-1])
	if err != nil {
		return models.ParsedModel{}, err
	}

	return models.ParsedModel{
		Name:         name,
		DependsOn:    depends,
		ParsedFields: fields,
	}, nil
}

func (p *parser) splitByInstructions(match []byte) []string {
	rawInstructions := p.InstructionRegex.Split(string(match), -1)
	instructions := make([]string, 0, len(rawInstructions))

	for _, instruction := range rawInstructions {
		i := strings.TrimSpace(instruction)
		if i == "" {
			continue
		}

		instructions = append(instructions, i)
	}

	return instructions
}

func (p *parser) parseFields(fields []string) ([]models.ParsedField, []string, error) {
	parsedFields := make([]models.ParsedField, len(fields))
	var depends []string
	for i, field := range fields {
		parsedField, err := p.parseField(field)
		if err != nil {
			return nil, nil, err
		}

		depends = append(depends, parsedField.DependsOn...)
		parsedFields[i] = parsedField
	}

	return parsedFields, depends, nil
}

func (p *parser) parseField(field string) (models.ParsedField, error) {
	splitField := strings.Split(field, ":")
	if len(splitField) != 2 {
		return models.ParsedField{}, errors.New("bad field format: " + field)
	}

	name, type_ := strings.TrimSpace(splitField[0]), strings.TrimSpace(splitField[1])
	type_, isArray := strings.CutSuffix(type_, "[]")
	name, nullable := strings.CutSuffix(name, "?")

	var depends []string
	types := strings.Split(type_, "|")
	for i, type_ := range types {
		if !models.PrimitivesList.Contain(type_) {
			depends = append(depends, type_)
		}

		types[i] = strings.TrimSpace(type_)
	}

	return models.ParsedField{
		Name:      name,
		DependsOn: depends,
		Types:     types,
		Nullable:  nullable,
		IsArray:   isArray,
	}, nil
}
