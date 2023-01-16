package common

import (
	"reflect"
	"unicode"
)

type ForEachField func(index int, field reflect.StructField) (breakIteration bool)

type TagRetriever func(tagName string) string

type Introspector struct {
	PtrType       reflect.Type
	Fields        []reflect.StructField
	fieldIndexMap map[string]int
}

// Iterate fields
func (it *Introspector) IterFields(forEach ForEachField) {
	for i, f := range it.Fields {
		if doBreak := forEach(i, f); doBreak {
			return
		}
	}
}

// Get tag retriever for a field
func (it *Introspector) TagRetriever(fieldName string) (t TagRetriever, isFieldFound bool) {
	f, ok := it.Field(fieldName)
	if !ok {
		return nil, false
	}

	t = func(tagName string) string {
		return f.Tag.Get(tagName)
	}
	isFieldFound = true
	return
}

// Get tag by of field
func (it *Introspector) Tag(fieldName string, tagName string) (tag string, isFieldFound bool) {
	f, ok := it.Field(fieldName)
	if !ok {
		return
	}

	tag = f.Tag.Get(tagName)
	isFieldFound = true
	return
}

// Get field by name
func (it *Introspector) Field(fieldName string) (field reflect.StructField, isFieldFound bool) {
	i, ok := it.fieldIndexMap[fieldName]
	if !ok {
		field = reflect.StructField{}
		return
	}

	field = it.Fields[i]
	isFieldFound = true
	return
}

// Create new Introspector
func Introspect(ptr any) Introspector {
	t := reflect.TypeOf(ptr)
	el := t
	if t.Kind() == reflect.Pointer {
		el = t.Elem()
	}
	fields := CollectTypeFields(t)
	indexMap := map[string]int{}

	for i, v := range fields {
		indexMap[v.Name] = i
	}

	return Introspector{PtrType: el, Fields: fields, fieldIndexMap: indexMap}
}

// Get Fields of A Type
func CollectFields(ptr any) []reflect.StructField {
	el := reflect.TypeOf(ptr).Elem()
	return CollectTypeFields(el)
}

// Get Fields of A Type
func CollectTypeFields(eleType reflect.Type) []reflect.StructField {
	fields := []reflect.StructField{}
	for i := 0; i < eleType.NumField(); i++ {
		fields = append(fields, eleType.Field(i))
	}
	return fields
}

// Check if field is exposed
func IsFieldExposed(fieldName string) bool {
	for _, c := range fieldName {
		return unicode.IsUpper(c) // only check first unicode character
	}
	return false
}
