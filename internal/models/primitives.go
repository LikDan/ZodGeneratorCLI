package models

import "slices"

type Primitives struct {
	list []string
}

var PrimitivesList = Primitives{
	list: []string{"number", "string", "boolean", "any", "null", "undefined", "DateTime"},
}

func (p *Primitives) Contain(type_ string) bool {
	return slices.Contains(p.list, type_)
}
