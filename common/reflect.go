package common

import (
	"reflect"
)

type TagRetriever func(tagName string) string

type Introspector struct {
	PtrType       reflect.Type
	Fields        []reflect.StructField
	fieldIndexMap map[string]int
}

// Get tag retriever for a field
func (th *Introspector) TagRetriever(fieldName string) (t TagRetriever, isFieldFound bool) {
	f, ok := th.Field(fieldName)
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
func (th *Introspector) Tag(fieldName string, tagName string) (tag string, isFieldFound bool) {
	f, ok := th.Field(fieldName)
	if !ok {
		return
	}

	tag = f.Tag.Get(tagName)
	isFieldFound = true
	return
}

// Get field by name
func (th *Introspector) Field(fieldName string) (field reflect.StructField, isFieldFound bool) {
	i, ok := th.fieldIndexMap[fieldName]
	if !ok {
		field = reflect.StructField{}
		return
	}

	field = th.Fields[i]
	isFieldFound = true
	return
}

// Create new Introspector, ptr must be a pointer
func Introspect(ptr any) Introspector {
	t := reflect.TypeOf(ptr).Elem()
	fields := CollectTypeFields(t)
	indexMap := map[string]int{}
	for i, v := range fields {
		indexMap[v.Name] = i
	}
	return Introspector{PtrType: t, Fields: fields, fieldIndexMap: indexMap}
}

// Get Fields of A Type, ptr must be a pointer
func CollectFields(ptr any) []reflect.StructField {
	el := reflect.TypeOf(ptr).Elem()
	return CollectTypeFields(el)
}

// Get Fields of A Type, ptr must be a pointer
func CollectTypeFields(eleType reflect.Type) []reflect.StructField {
	fields := []reflect.StructField{}
	for i := 0; i < eleType.NumField(); i++ {
		fields = append(fields, eleType.Field(i))
	}
	return fields
}
