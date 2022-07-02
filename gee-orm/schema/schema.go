package schema

import (
	"geeorm/dialect"
	"reflect"
)

type Field struct {
	Name string
	Type string
	Tag  string
}

type Schema struct {
	Model      interface{}
	Name       string
	Fields     []*Field
	FieldNames []string
	fieldMap   map[string]*Field
}

func (s *Schema) GetFields(name string) *Field {
	return s.fieldMap[name]
}

// Parse 传入的是指针，所以需要用reflect.Indirect()获得指针指向的实例
func Parse(dest interface{}, d dialect.Dialect) *Schema {
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	schema := &Schema{
		Model:    dest,
		Name:     modelType.Name(),
		fieldMap: make(map[string]*Field),
	}
	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)
		field := &Field{
			Name: p.Name,
			Type: d.DataTypeOf(reflect.Indirect(reflect.New(p.Type))),
		}
		if v, ok := p.Tag.Lookup("geeorm"); ok {
			field.Tag = v
		}
		schema.Fields = append(schema.Fields, field)
		schema.FieldNames = append(schema.FieldNames, p.Name)
		schema.fieldMap[p.Name] = field
	}
	return schema
}

func (s *Schema) RecordValues(dest interface{}) []interface{} {
	destValue := reflect.Indirect(reflect.ValueOf(dest))
	var fieldsValues []interface{}
	for _, field := range s.Fields {
		fieldsValues = append(fieldsValues, destValue.FieldByName(field.Name).Interface())
	}
	return fieldsValues
}
