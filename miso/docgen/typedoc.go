package docgen

import (
	"go/types"
	"reflect"
	"strings"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/json"
)

// BuildTypeDescFromType builds a TypeDesc from a go/types.Type, mirroring
// miso.BuildTypeDesc(reflect.Value) but using go/types instead of reflect.
func BuildTypeDescFromType(t types.Type, pkg *types.Package) miso.TypeDesc {
	qual := types.RelativeTo(pkg)

	switch typ := t.(type) {
	case *types.Pointer:
		elemDesc := BuildTypeDescFromType(typ.Elem(), pkg)
		elemDesc.IsPtr = true
		if elemDesc.IsSlice {
			elemDesc.IsPtrSlice = true
		}
		return elemDesc

	case *types.Named:
		typeName := typ.Obj().Name()
		var typePkg string
		if typ.Obj().Pkg() != nil {
			typePkg = typ.Obj().Pkg().Path()
		}

		underlying := typ.Underlying()

		// Handle type alias chain: underlying is itself *types.Named
		if namedUnder, ok := underlying.(*types.Named); ok {
			desc := BuildTypeDescFromType(namedUnder, pkg)
			desc.TypeName = typeName
			desc.TypePkg = typePkg
			return desc
		}

		if st, ok := underlying.(*types.Struct); ok {
			seen := make(map[types.Type]bool)
			seen[typ] = true
			fields := extractFieldDescs(st, pkg, seen)
			return miso.TypeDesc{
				TypeName: typeName,
				TypePkg:  typePkg,
				Fields:   fields,
			}
		}

		// Other underlying types (basic, slice, map, etc.)
		return miso.TypeDesc{
			TypeName: typeName,
			TypePkg:  typePkg,
		}

	case *types.Slice:
		elem := typ.Elem()

		isPtrElem := false
		inner := elem
		if ptr, ok := inner.(*types.Pointer); ok {
			isPtrElem = true
			inner = ptr.Elem()
		}

		if named, ok := inner.(*types.Named); ok {
			if st, ok := named.Underlying().(*types.Struct); ok {
				seen := make(map[types.Type]bool)
				fields := extractFieldDescs(st, pkg, seen)
				desc := miso.TypeDesc{
					IsSlice:    true,
					IsSlicePtr: isPtrElem,
					Fields:     fields,
					TypeName:   named.Obj().Name(),
				}
				if named.Obj().Pkg() != nil {
					desc.TypePkg = named.Obj().Pkg().Path()
				}
				return desc
			}
		}

		// Slice of basic types or unnamed types
		return miso.TypeDesc{
			IsSlice:  true,
			TypeName: types.TypeString(elem, qual),
		}

	case *types.Struct:
		seen := make(map[types.Type]bool)
		fields := extractFieldDescs(typ, pkg, seen)
		return miso.TypeDesc{
			Fields: fields,
		}

	case *types.Basic:
		return miso.TypeDesc{
			TypeName: typ.Name(),
		}

	case *types.Map:
		return miso.TypeDesc{
			TypeName: types.TypeString(typ, qual),
		}

	default:
		return miso.TypeDesc{
			TypeName: types.TypeString(typ, qual),
		}
	}
}

// extractFieldDescs extracts field descriptions from a go/types struct.
// The seen map is used for cycle detection.
func extractFieldDescs(st *types.Struct, pkg *types.Package, seen map[types.Type]bool) []miso.FieldDesc {
	qual := types.RelativeTo(pkg)
	fields := make([]miso.FieldDesc, 0, st.NumFields())

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Exported() {
			continue
		}

		tag := st.Tag(i)
		stTag := reflect.StructTag(tag)

		// Skip json:"-"
		if jv := stTag.Get("json"); jv == "-" {
			continue
		}

		// Skip form/header tags
		if hasIgnoredTag(tag) {
			continue
		}

		// Parse json name
		jsonName, jsonTag := parseJsonName(tag, field.Name())
		if jsonTag == "" {
			jsonTag = jsonName
		}

		// OriginTypeName: TypeString relative to pkg. Keep * and [] prefixes —
		// they are stripped later by PureGoTypeName() when needed. This mirrors
		// rfutil.TypeName which preserves pointer/slice prefixes.
		originTypeName := types.TypeString(field.Type(), qual)

		// OriginTypeNameWithPkg: use package name qualifier to match reflect.Type.String().
		// Always include the package name (even for the handler's own package) because
		// reflect.Type.String() never suppresses same-package qualifiers.
		pkgNameQual := func(other *types.Package) string {
			return other.Name()
		}
		originTypeNameWithPkg := types.TypeString(field.Type(), pkgNameQual)

		// TypePkg: recursively unwrap to find package path
		tp := typePkgPath(field.Type())

		// TypeNameAlias: look up in ApiDocTypeAlias. Use PureGoTypeName for
		// alias lookup (strips pkg prefix like "atom."), but keep originTypeName
		// for display (preserves * and []).
		typeNameAlias, typeAliasMatched := resolveTypeAlias(originTypeName, originTypeNameWithPkg, miso.PureGoTypeName(originTypeName))

		// Valid tag
		validTag := stTag.Get("valid")
		if validTag == "" {
			validTag = stTag.Get("validate")
		}

		// Desc tag
		descTag := stTag.Get("desc")
		if descTag == "" {
			if xd := stTag.Get("xdesc"); xd != "" {
				descTag = xd
			}
		}

		jd := miso.FieldDesc{
			GoFieldName:           field.Name(),
			JsonName:              jsonName,
			TypeNameAlias:         typeNameAlias,
			TypePkg:               tp,
			OriginTypeName:        originTypeName,
			OriginTypeNameWithPkg: originTypeNameWithPkg,
			DescTag:               descTag,
			ValidTag:              validTag,
			JsonTag:               jsonTag,
			Fields:                []miso.FieldDesc{},
		}

		// Set type flags
		switch ft := field.Type().(type) {
		case *types.Slice:
			jd.IsSliceOrArray = true
			if _, ok := ft.Elem().(*types.Pointer); ok {
				jd.IsSliceOfPointer = true
			}
		case *types.Array:
			jd.IsSliceOrArray = true
		case *types.Map:
			jd.IsMap = true
		case *types.Pointer:
			jd.IsPointer = true
		}

		// When IsPointer is true but TypeNameAlias doesn't have "*" prefix
		// (alias resolved to base type), prepend "*" to match runtime behavior.
		// Only for scalar/primitives — skip slice/map aliases (e.g. Set[T] → []T)
		// where the pointer is semantically redundant.
		if jd.IsPointer && !strings.HasPrefix(jd.TypeNameAlias, "*") {
			if !strings.HasPrefix(jd.TypeNameAlias, "[]") &&
				!strings.HasPrefix(jd.TypeNameAlias, "map[") {
				jd.TypeNameAlias = "*" + jd.TypeNameAlias
			}
		}

		// Recurse into nested structs (skip if type alias matched)
		if !typeAliasMatched && !seen[field.Type()] {
			seen[field.Type()] = true
			nested := extractNestedFields(field.Type(), pkg, seen)
			jd.Fields = append(jd.Fields, nested...)
		}

		fields = append(fields, jd)
	}

	return fields
}

// extractNestedFields extracts field descriptions from a nested struct type.
// Returns nil if the type does not contain a struct.
func extractNestedFields(t types.Type, pkg *types.Package, seen map[types.Type]bool) []miso.FieldDesc {
	switch typ := t.(type) {
	case *types.Named:
		if st, ok := typ.Underlying().(*types.Struct); ok {
			seen[typ] = true
			return extractFieldDescs(st, pkg, seen)
		}

	case *types.Pointer:
		if named, ok := typ.Elem().(*types.Named); ok {
			if !seen[named] {
				if st, ok := named.Underlying().(*types.Struct); ok {
					seen[named] = true
					return extractFieldDescs(st, pkg, seen)
				}
			}
		}

	case *types.Slice:
		elem := typ.Elem()
		if ptr, ok := elem.(*types.Pointer); ok {
			elem = ptr.Elem()
		}
		if named, ok := elem.(*types.Named); ok {
			if !seen[named] {
				if st, ok := named.Underlying().(*types.Struct); ok {
					seen[named] = true
					return extractFieldDescs(st, pkg, seen)
				}
			}
		}

	case *types.Interface:
		// Skip interface types — cannot extract struct fields
	}

	return nil
}

// hasIgnoredTag checks if the tag has a non-empty value for "form" or "header".
func hasIgnoredTag(tag string) bool {
	st := reflect.StructTag(tag)
	if v := st.Get("form"); v != "" {
		return true
	}
	if v := st.Get("header"); v != "" {
		return true
	}
	return false
}

// parseJsonName extracts the JSON field name and raw JSON tag value from a struct tag.
func parseJsonName(tag string, fieldName string) (jsonName string, jsonTag string) {
	st := reflect.StructTag(tag)
	jsonTag = st.Get("json")
	if jsonTag != "" {
		tokz := strings.TrimSpace(strings.Split(jsonTag, ",")[0])
		if tokz == "" {
			// e.g., ',omitEmpty'
			jsonName = json.NamingStrategyTranslate(fieldName)
		} else {
			jsonName = tokz
		}
	} else {
		jsonName = json.NamingStrategyTranslate(fieldName)
	}
	return
}

// typePkgPath recursively unwraps pointer/slice/array to get the package path
// of the underlying named type.
func typePkgPath(t types.Type) string {
	switch typ := t.(type) {
	case *types.Pointer:
		return typePkgPath(typ.Elem())
	case *types.Slice:
		return typePkgPath(typ.Elem())
	case *types.Array:
		return typePkgPath(typ.Elem())
	case *types.Named:
		if typ.Obj().Pkg() != nil {
			return typ.Obj().Pkg().Path()
		}
	}
	return ""
}

// resolveTypeAlias looks up the type name in ApiDocTypeAlias, trying multiple forms.
// Returns the alias string and whether a match was found.
// rawOrigin: full module path from types.RelativeTo (e.g., "github.com/.../money.Amt")
// pkgOrigin: short package name form (e.g., "money.Amt") matching reflect.Type.String()
// pureOrigin: bare type name with generics/pointers stripped (e.g., "Amt")
func resolveTypeAlias(rawOrigin string, pkgOrigin string, pureOrigin string) (string, bool) {
	// Try exact raw match (e.g., "*atom.Time")
	if v, ok := miso.ApiDocTypeAlias[rawOrigin]; ok {
		return v, true
	}

	// Try with "*" prefix added
	if v, ok := miso.ApiDocTypeAlias["*"+rawOrigin]; ok {
		return v, true
	}

	// Try short package name form (e.g., "money.Amt") — matches how middleware
	// register aliases in init() via reflect.Type.String()
	if v, ok := miso.ApiDocTypeAlias[pkgOrigin]; ok {
		return v, true
	}
	if v, ok := miso.ApiDocTypeAlias["*"+pkgOrigin]; ok {
		return v, true
	}

	// Try pure (stripped) match (e.g., "Time")
	if v, ok := miso.ApiDocTypeAlias[pureOrigin]; ok {
		return v, true
	}

	// Try "*" + pure (e.g., "*Time")
	if v, ok := miso.ApiDocTypeAlias["*"+pureOrigin]; ok {
		return v, true
	}

	// Try raw without leading "*"
	trimmed := strings.TrimPrefix(rawOrigin, "*")
	if trimmed != rawOrigin {
		if v, ok := miso.ApiDocTypeAlias[trimmed]; ok {
			return v, true
		}
	}

	// Try with full package path prefix stripped, preserving generics, pointers, etc.
	// types.TypeString with RelativeTo uses full package paths (e.g.,
	// "github.com/curtisnewbie/miso/util/hash.Set[string]"), but the alias map keys
	// use bare type names ("Set[string]") or short pkg names ("hash.Set[string]").
	if i := strings.LastIndexByte(rawOrigin, '.'); i >= 0 {
		noPkg := rawOrigin[i+1:]
		if noPkg != rawOrigin {
			if v, ok := miso.ApiDocTypeAlias[noPkg]; ok {
				return v, true
			}
			if v, ok := miso.ApiDocTypeAlias["*"+noPkg]; ok {
				return v, true
			}
		}
	}

	// Fallback: match rfutil.TypeName behavior.
	// - Pointer/slice types → pkg-qualified (like reflect.Type.String())
	// - Named value types → bare name (like reflect.Type.Name())
	if strings.HasPrefix(rawOrigin, "*") || strings.HasPrefix(rawOrigin, "[]") {
		return pkgOrigin, false
	}
	return pureOrigin, false
}
