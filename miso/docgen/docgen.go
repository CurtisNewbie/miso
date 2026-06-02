package docgen

import (
	"fmt"
	"go/types"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/miso/sourceparser"
	"github.com/curtisnewbie/miso/util/pair"
	"github.com/curtisnewbie/miso/util/slutil"
	"golang.org/x/tools/go/packages"
)

// SourceFile represents a source file to parse for endpoint registrations.
type SourceFile struct {
	Path string
}

// Logger is the minimal logging interface used by the doc generator.
type Logger interface {
	Infof(format string, args ...any)
	Debugf(format string, args ...any)
}

// nopLogger discards all log messages.
type nopLogger struct{}

func (nopLogger) Infof(string, ...any)  {}
func (nopLogger) Debugf(string, ...any) {}

// log is the package-level logger, set by BuildManualRouteDocs.
var log Logger = nopLogger{}

// cached miso package loader, reused by resolveGenericTypeRef and buildRespTypeDesc
var (
	misoPkgOnce  sync.Once
	misoPkgCache *types.Package
	misoPkgErr   error
)

// lookupTypeInPkg looks up a type name in the package scope. If not found and the
// name contains a '.' (e.g., "api.PostRes"), it splits by the last '.' and tries
// the lookup in imported packages.
func lookupTypeInPkg(pkg *types.Package, typeName string) types.Object {
	obj := pkg.Scope().Lookup(typeName)
	if obj != nil {
		return obj
	}
	if dotIdx := strings.LastIndexByte(typeName, '.'); dotIdx >= 0 {
		pkgName := typeName[:dotIdx]
		typeOnly := typeName[dotIdx+1:]
		for _, imp := range pkg.Imports() {
			if imp.Name() == pkgName {
				return imp.Scope().Lookup(typeOnly)
			}
		}
	}
	return nil
}

// resolveTypeRef resolves a structured TypeRef (from the sourceparser) into a full TypeDesc.
func resolveTypeRef(ref sourceparser.TypeRef, pkg *types.Package) miso.TypeDesc {
	var desc miso.TypeDesc

	lookupName := ref.Name
	if ref.PkgName != "" {
		lookupName = ref.PkgName + "." + ref.Name
	}

	// Try generic resolution first when type args are present (e.g., miso.PageRes[PostRes]).
	// Uses the structured TypeRef directly — no string round-trip.
	if len(ref.TypeArgs) > 0 {
		if d, ok := resolveGenericTypeRef(ref.PkgName, ref.Name, ref.TypeArgs, pkg); ok {
			desc = d
		} else {
			desc = miso.TypeDesc{TypeName: ref.String()}
		}
	} else if obj := lookupTypeInPkg(pkg, lookupName); obj != nil {
		// Direct/cross-package lookup for non-generic types
		desc = BuildTypeDescFromType(obj.Type(), pkg)
	} else if isBuiltInGo(ref.Name) {
		desc = miso.TypeDesc{TypeName: ref.Name}
	} else {
		// Fallback: try PureGoTypeName
		baseName := miso.PureGoTypeName(lookupName)
		if baseName != lookupName {
			if obj := lookupTypeInPkg(pkg, baseName); obj != nil {
				desc = BuildTypeDescFromType(obj.Type(), pkg)
			} else {
				desc = miso.TypeDesc{TypeName: ref.FullString()}
			}
		} else {
			desc = miso.TypeDesc{TypeName: ref.FullString()}
		}
	}

	applyFlags(&desc, ref.IsPtr, ref.IsSlice, ref.IsSlicePtr)
	return desc
}

// isBuiltInGo reports whether name is a Go built-in type.
func isBuiltInGo(name string) bool {
	switch name {
	case "bool", "byte", "complex64", "complex128",
		"error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64",
		"rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	}
	return false
}

func applyFlags(desc *miso.TypeDesc, isPtr, isSlice, isSlicePtr bool) {
	if isPtr {
		desc.IsPtr = true
	}
	if isSlice {
		desc.IsSlice = true
	}
	if isSlicePtr {
		desc.IsSlicePtr = true
	}
}

// loadMisoPackage loads the miso package and caches the result for reuse.
func loadMisoPackage() (*types.Package, error) {
	misoPkgOnce.Do(func() {
		cfg := &packages.Config{Mode: packages.NeedTypes}
		pkgs, err := packages.Load(cfg, "github.com/curtisnewbie/miso/miso")
		if err != nil || len(pkgs) == 0 || pkgs[0].Types == nil {
			misoPkgErr = fmt.Errorf("failed to load miso package: %w", err)
			return
		}
		misoPkgCache = pkgs[0].Types
	})
	return misoPkgCache, misoPkgErr
}

// resolveGenericTypeRef resolves a generic type from its structured parts.
// Uses go/types to instantiate the base type with the given type arguments.
func resolveGenericTypeRef(pkgName, typeName string, typeArgs []sourceparser.TypeRef, pkg *types.Package) (miso.TypeDesc, bool) {
	// Find the base type's package
	var basePkg *types.Package
	if pkgName == "" || pkgName == pkg.Name() {
		basePkg = pkg
	} else {
		for _, imp := range pkg.Imports() {
			if imp.Name() == pkgName {
				basePkg = imp
				break
			}
		}
		if basePkg == nil {
			misoPkg, err := loadMisoPackage()
			if err == nil && misoPkg.Name() == pkgName {
				basePkg = misoPkg
			}
		}
		if basePkg == nil {
			return miso.TypeDesc{}, false
		}
	}

	// Find the base generic type
	baseObj := basePkg.Scope().Lookup(typeName)
	if baseObj == nil {
		return miso.TypeDesc{}, false
	}

	baseNamed, ok := baseObj.Type().(*types.Named)
	if !ok {
		return miso.TypeDesc{}, false
	}

	// Verify type param count matches
	tparams := baseNamed.TypeParams()
	if tparams == nil || tparams.Len() != len(typeArgs) {
		return miso.TypeDesc{}, false
	}

	// Resolve each type argument from TypeRef
	targs := make([]types.Type, len(typeArgs))
	for i, argRef := range typeArgs {
		argType, ok := resolveTypeArgFromRef(argRef, pkg)
		if !ok {
			return miso.TypeDesc{}, false
		}
		targs[i] = argType
	}

	// Instantiate the generic type
	inst, err := types.Instantiate(nil, baseNamed, targs, false)
	if err != nil {
		return miso.TypeDesc{}, false
	}

	fullName := pkgName
	if fullName != "" {
		fullName += "."
	}
	fullName += typeName
	fullName += "["
	for i, a := range typeArgs {
		if i > 0 {
			fullName += ","
		}
		fullName += a.String()
	}
	fullName += "]"

	desc := BuildTypeDescFromType(inst, pkg)
	desc.TypeName = fullName
	return desc, true
}

// resolveTypeArgFromRef resolves a TypeRef to a types.Type by looking it up
// in the handler's package scope or imported packages.
func resolveTypeArgFromRef(ref sourceparser.TypeRef, pkg *types.Package) (types.Type, bool) {
	name := ref.Name
	if ref.PkgName != "" {
		name = ref.PkgName + "." + name
	}
	if obj := lookupTypeInPkg(pkg, name); obj != nil {
		return obj.Type(), true
	}
	return nil, false
}

// buildRespTypeDesc loads the Resp struct from the miso package and builds a TypeDesc
// with the Data field replaced by the given DTO TypeDesc.
// Falls back to a minimal hardcoded Resp if the miso package cannot be loaded.
func buildRespTypeDesc(dto miso.TypeDesc) miso.TypeDesc {
	misoPkg, err := loadMisoPackage()
	if err != nil {
		log.Infof("Failed to load miso package for Resp type: %v, using fallback", err)
		return fallbackResp(dto)
	}

	obj := misoPkg.Scope().Lookup("Resp")
	if obj == nil {
		log.Infof("Resp type not found in miso package, using fallback")
		return fallbackResp(dto)
	}

	// Build TypeDesc for Resp from actual go/types
	respDesc := BuildTypeDescFromType(obj.Type(), misoPkg)

	// If DTO is any/interface{}, exclude the Data field (matching runtime behavior
	// where interface{} with nil value is skipped)
	if dto.TypeName == "any" || dto.TypeName == "interface{}" {
		filtered := make([]miso.FieldDesc, 0, len(respDesc.Fields)-1)
		for _, f := range respDesc.Fields {
			if f.GoFieldName != "Data" {
				filtered = append(filtered, f)
			}
		}
		respDesc.Fields = filtered
		return respDesc
	}

	// Replace the Data field with the DTO's concrete type info
	for i, f := range respDesc.Fields {
		if f.GoFieldName == "Data" {
			dataField := buildDataField(dto)
			// Preserve the original DescTag from the Resp struct if available
			if f.DescTag != "" {
				dataField.DescTag = f.DescTag
			}
			respDesc.Fields[i] = dataField
			break
		}
	}

	return respDesc
}

// fallbackResp returns a minimal hardcoded Resp when the real type can't be loaded.
func fallbackResp(dto miso.TypeDesc) miso.TypeDesc {
	fields := []miso.FieldDesc{
		{GoFieldName: "ErrorCode", JsonName: "errorCode", TypeNameAlias: "string", OriginTypeName: "string", DescTag: "error code"},
		{GoFieldName: "Msg", JsonName: "msg", TypeNameAlias: "string", OriginTypeName: "string", DescTag: "message"},
		{GoFieldName: "Error", JsonName: "error", TypeNameAlias: "bool", OriginTypeName: "bool", DescTag: "whether the request was successful"},
	}
	if dto.TypeName != "any" && dto.TypeName != "interface{}" {
		fields = append(fields, buildDataField(dto))
	}
	return miso.TypeDesc{
		TypeName: "Resp",
		Fields:   fields,
	}
}

// buildDataField creates the Data field descriptor from a DTO TypeDesc.
func buildDataField(dto miso.TypeDesc) miso.FieldDesc {
	typeName := dto.TypeName

	// Only add package name prefix for pointer/slice types, matching the
	// behavior of reflect.Type.String(). For plain named types (value,
	// without * or []), use the bare name like reflect.Type.Name().
	if (dto.IsSlice || dto.IsSlicePtr || dto.IsPtr) && !strings.ContainsRune(typeName, '.') {
		if pkgName := path.Base(dto.TypePkg); pkgName != "" && pkgName != "." {
			typeName = pkgName + "." + typeName
		}
	}

	// Add pointer/slice prefixes in correct order: inner-most first.
	// IsSlicePtr: slice of pointers ([]*T) → add * before [].
	if dto.IsSlicePtr {
		typeName = "*" + typeName
	}
	if dto.IsSlice {
		typeName = "[]" + typeName
	}
	if dto.IsPtr {
		typeName = "*" + typeName
	}

	fd := miso.FieldDesc{
		GoFieldName:    "Data",
		JsonName:       "data",
		TypeNameAlias:  typeName,
		OriginTypeName: typeName,
		TypePkg:        dto.TypePkg,
		DescTag:        "response data",
		Fields:         dto.Fields,
	}
	if strings.HasPrefix(dto.TypeName, "map[") {
		fd.IsMap = true
	}
	if dto.IsSlice {
		fd.IsSliceOrArray = true
	}
	if dto.IsPtr {
		fd.IsPointer = true
	}
	if dto.IsSlicePtr {
		fd.IsSliceOfPointer = true
	}
	return fd
}

// BuildManualRouteDocs parses all Go files for manual endpoint registrations
// and builds HttpRouteDoc objects using go/types type resolution.
func BuildManualRouteDocs(files []SourceFile, modName string, l Logger) []miso.HttpRouteDoc {
	if l != nil {
		log = l
	}
	// Filter out generated and test files
	files = slutil.Filter(files, func(f SourceFile) bool {
		base := path.Base(f.Path)
		return !strings.HasSuffix(base, "_test.go")
	})

	// Group parsed endpoints by directory
	dirMap := make(map[string][]*sourceparser.ParsedEndpoint)

	for _, f := range files {
		dir := path.Dir(f.Path)
		eps, err := sourceparser.ParseFile(f.Path)
		if err != nil {
			log.Debugf("sourceparser.ParseFile(%s) failed: %v", f.Path, err)
			continue
		}
		if len(eps) > 0 {
			dirMap[dir] = append(dirMap[dir], eps...)
		}
	}

	var allDocs []miso.HttpRouteDoc

	// Sort directory keys for deterministic output order
	dirs := make([]string, 0, len(dirMap))
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		eps := dirMap[dir]
		pkgPath := modName + "/" + dir
		pkgPath = strings.TrimRight(pkgPath, "/")

		pkg, err := loadPackageFromDir(pkgPath, dir)
		if err != nil {
			log.Debugf("Failed to load package %s: %v", pkgPath, err)
			continue
		}

		for _, ep := range eps {
			doc := miso.HttpRouteDoc{
				Name:     ep.FuncName,
				Url:      ep.URL,
				Method:   ep.Method,
				Desc:     ep.Desc,
				Scope:    ep.Scope,
				Resource: ep.Resource,
			}

			// Query params
			for _, q := range ep.QueryParams {
				doc.QueryParams = append(doc.QueryParams, miso.ParamDoc{Name: q.Left, Desc: q.Right})
			}

			// Headers
			for _, h := range ep.Headers {
				doc.Headers = append(doc.Headers, miso.ParamDoc{Name: h.Left, Desc: h.Right})
			}

			// Request type
			if ep.RequestRef != nil {
				doc.JsonRequestDesc = resolveTypeRef(*ep.RequestRef, pkg)
			}

			// Extract query/header params from DocQueryReq/DocHeaderReq structs.
			// These structs use "query"/"form" tags for query params and "header"
			// tags for header params.
			extractRequestParams(&doc, ep, pkg)

			// Response type — RawHandler with DocJsonResp gets the DTO as-is
			// (no Resp wrapping), matching runtime where DocJsonResp stores
			// the raw value without PayloadJsonBuilder wrapping.
			if ep.ResponseRef != nil {
				ref := *ep.ResponseRef
				if ref.Name == "any" && ref.PkgName == "" {
					// Handler returns only error → show Resp envelope without Data field
					doc.JsonResponseDesc = buildRespTypeDesc(miso.TypeDesc{TypeName: "any"})
				} else {
					desc := resolveTypeRef(ref, pkg)
					if desc.TypeName != "" && ep.Handler != "RawHandler" {
						desc = buildRespTypeDesc(desc)
					}
					doc.JsonResponseDesc = desc
				}
			}

			// Angular NgTable demo for pageable endpoints
			if hasExtra(ep.Extras, miso.ExtraNgTable) {
				doc.NgTableDemo = miso.GenNgTableDemo(doc)
			}

			allDocs = append(allDocs, doc)
		}
	}

	return allDocs
}

// extractRequestParams extracts query and header params from DocQueryReq/DocHeaderReq struct types.
// Fields with "form" struct tag become query params; fields with "header" tags become header params.
func extractRequestParams(doc *miso.HttpRouteDoc, ep *sourceparser.ParsedEndpoint, pkg *types.Package) {
	if ep.QueryReqType != "" {
		params := lookupTagParams(ep.QueryReqType, pkg, miso.TagQueryParam)
		doc.QueryParams = append(doc.QueryParams, params...)
	}
	if ep.HeaderReqType != "" {
		params := lookupTagParams(ep.HeaderReqType, pkg, miso.TagHeaderParam)
		doc.Headers = append(doc.Headers, params...)
	}
}

// lookupTagParams resolves a type by name in the package scope and extracts ParamDoc
// entries from struct fields that have the specified struct tag.
func lookupTagParams(typeName string, pkg *types.Package, tagKey string) []miso.ParamDoc {
	// Strip pointer/slice prefixes to get the base type name for lookup
	lookupName := typeName
	if strings.HasPrefix(lookupName, "*") {
		lookupName = lookupName[1:]
	}
	if strings.HasPrefix(lookupName, "[]") {
		lookupName = lookupName[2:]
	}

	obj := lookupTypeInPkg(pkg, lookupName)
	if obj == nil {
		return nil
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}

	var params []miso.ParamDoc
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Exported() {
			continue
		}
		tag := reflect.StructTag(st.Tag(i))
		v := tag.Get(tagKey)
		if v == "" {
			continue
		}
		desc := tag.Get("desc")
		if desc == "" {
			desc = tag.Get("xdesc")
		}
		params = append(params, miso.ParamDoc{Name: v, Desc: desc})
	}
	return params
}

// The sourceparser stores AST selector expressions (e.g., "miso.ExtraNgTable"),
// but the runtime constant values use a different format (e.g., "miso-NgTable").
// This function converts the selector form to the constant form for comparison.
func hasExtra(extras []pair.StrPair, extraConst string) bool {
	for _, ex := range extras {
		// Sourceparser stores "miso.ExtraXxx", runtime uses "miso-Xxx"
		if strings.HasPrefix(ex.Left, "miso.Extra") {
			resolved := "miso-" + strings.TrimPrefix(ex.Left, "miso.Extra")
			if resolved == extraConst {
				return true
			}
		}
	}
	return false
}

// loadPackageFromDir loads a Go package with the specified working directory.
func loadPackageFromDir(pkgPath string, dir string) (*types.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedName | packages.NeedImports | packages.NeedDeps,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, err
	}
	if len(pkgs) > 0 && pkgs[0].Types != nil {
		return pkgs[0].Types, nil
	}
	return nil, fmt.Errorf("no types loaded for %s", pkgPath)
}
