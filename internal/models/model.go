package models

type ParsedFile struct {
	Path   string
	Models []ParsedModel
}

type ParsedModel struct {
	Path         string
	Name         string
	DependsOn    []string
	ParsedFields []ParsedField
}

type ParsedField struct {
	Name      string
	DependsOn []string
	Types     []string
	Nullable  bool
	IsArray   bool
}
