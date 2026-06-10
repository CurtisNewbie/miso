package sourceparser

import (
	"fmt"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"github.com/curtisnewbie/miso/util/pair"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

var httpMethodMap = map[string]string{
	"HttpPost":    "POST",
	"HttpGet":     "GET",
	"HttpPut":     "PUT",
	"HttpDelete":  "DELETE",
	"HttpPatch":   "PATCH",
	"HttpHead":    "HEAD",
	"HttpOptions": "OPTIONS",
	"HttpConnect": "CONNECT",
	"HttpTrace":   "TRACE",
	"HttpAny":     "ANY",
}

// ParsedEndpoint holds all metadata extracted from a miso endpoint registration chain.
type ParsedEndpoint struct {
	Method        string // POST, GET, etc.
	URL           string // "/api/v1"
	Handler       string // "AutoHandler", "ResHandler", "RawHandler", "Direct"
	Desc          string
	Scope         string
	Resource      string
	QueryReqType  string // type from DocQueryReq(struct{}) — struct with "query"/"form" tags
	HeaderReqType string // type from DocHeaderReq(struct{}) — struct with "header" tags
	QueryParams   []pair.StrPair
	Headers       []pair.StrPair
	Extras        []pair.StrPair
	FuncName      string   // from Extra(miso.ExtraName, ...) if present
	RequestRef    *TypeRef // request type
	ResponseRef   *TypeRef // response type
}

// ParsedPipeline holds metadata extracted from a rabbit.NewEventPipeline declaration chain.
type ParsedPipeline struct {
	Name       string   // from Document(name, ...)
	Desc       string   // from Document(_, desc, ...)
	Provider   string   // from Document(_, _, provider)
	Queue      string   // from NewEventPipeline[T](queue)
	MaxRetry   int      // from MaxRetry(n), -1 if not set
	LogPayload bool     // from LogPayload()
	PayloadRef *TypeRef // the type parameter T from NewEventPipeline[T]
}

// TypeRef is a structured representation of a Go type expression, preserving
// type wrappers (pointer, slice) that are lost in string round-trips.
type TypeRef struct {
	PkgName    string    // package import name (e.g., "miso", "api"), empty for built-ins
	Name       string    // type name (e.g., "Inbound", "PageRes", "string")
	IsPtr      bool      // *T
	IsSlice    bool      // []T
	IsSlicePtr bool      // []*T
	TypeArgs   []TypeRef // for generic types like miso.PageRes[PostRes]
	IsMap      bool      // map[K]V
	MapKey     *TypeRef  // key type for map
	MapValue   *TypeRef  // value type for map
}

// String reconstructs the type string from the structured representation.
func (t TypeRef) String() string {
	switch {
	case t.IsMap:
		k := ""
		if t.MapKey != nil {
			k = t.MapKey.String()
		}
		v := ""
		if t.MapValue != nil {
			v = t.MapValue.String()
		}
		return "map[" + k + "]" + v
	default:
		s := ""
		if t.PkgName != "" {
			s = t.PkgName + "."
		}
		s += t.Name
		if len(t.TypeArgs) > 0 {
			args := make([]string, len(t.TypeArgs))
			for i, a := range t.TypeArgs {
				args[i] = a.String()
			}
			s += "[" + strings.Join(args, ",") + "]"
		}
		return s
	}
}

// FullString returns the full type string including pointer/slice wrappers.
// This is the string that exprToString previously produced.
func (t TypeRef) FullString() string {
	base := t.String()
	if t.IsSlicePtr {
		base = "*" + base
	}
	if t.IsSlice {
		base = "[]" + base
	}
	if t.IsPtr {
		base = "*" + base
	}
	return base
}

// ParseFile parses a single Go file and extracts all miso endpoint registrations.
func ParseFile(filePath string) ([]*ParsedEndpoint, error) {
	f, err := decorator.ParseFile(nil, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	constVars := collectConstVars(f)

	var endpoints []*ParsedEndpoint
	dstutil.Apply(f, func(c *dstutil.Cursor) bool {
		call, ok := c.Node().(*dst.CallExpr)
		if !ok {
			return true
		}

		// Only process if this is the top of a chain
		// (parent is not a SelectorExpr which would mean chaining)
		if _, ok := c.Parent().(*dst.SelectorExpr); ok {
			return true
		}

		// Handle BaseRoute("/prefix").Group(endpoints...) pattern
		if eps := processGroupCall(call, constVars, nil); len(eps) > 0 {
			for _, ep := range eps {
				extractFuncName(ep)
				endpoints = append(endpoints, ep)
			}
			return false // Don't recurse into Group's children (already processed)
		}

		ep := analyzeCallChain(call, constVars, nil)
		if ep != nil {
			extractFuncName(ep)
			endpoints = append(endpoints, ep)
		}
		return true
	}, nil)

	return endpoints, nil
}

// ParseFileDst extracts all miso endpoint registrations from a pre-parsed *dst.File.
// extraConsts (optional) provides const/var values from other files in the package.
func ParseFileDst(f *dst.File, pkg *types.Package, extraConsts ...map[string]string) []*ParsedEndpoint {
	constVars := collectConstVars(f)
	if len(extraConsts) > 0 && extraConsts[0] != nil {
		for k, v := range extraConsts[0] {
			if _, exists := constVars[k]; !exists {
				constVars[k] = v
			}
		}
	}

	var endpoints []*ParsedEndpoint
	dstutil.Apply(f, func(c *dstutil.Cursor) bool {
		call, ok := c.Node().(*dst.CallExpr)
		if !ok {
			return true
		}
		if _, ok := c.Parent().(*dst.SelectorExpr); ok {
			return true
		}
		if eps := processGroupCall(call, constVars, pkg); len(eps) > 0 {
			for _, ep := range eps {
				extractFuncName(ep)
				endpoints = append(endpoints, ep)
			}
			return false
		}
		ep := analyzeCallChain(call, constVars, pkg)
		if ep != nil {
			extractFuncName(ep)
			endpoints = append(endpoints, ep)
		}
		return true
	}, nil)
	return endpoints
}

// ParsePipelines parses a single Go file and extracts all rabbit.NewEventPipeline declarations.
func ParsePipelines(filePath string) ([]*ParsedPipeline, error) {
	f, err := decorator.ParseFile(nil, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var pipelines []*ParsedPipeline
	dstutil.Apply(f, func(c *dstutil.Cursor) bool {
		call, ok := c.Node().(*dst.CallExpr)
		if !ok {
			return true
		}
		// Only process if this is the top of a chain
		// (parent is not a SelectorExpr which would mean chaining)
		if _, ok := c.Parent().(*dst.SelectorExpr); ok {
			return true
		}

		pp := analyzePipelineChain(call)
		if pp != nil {
			if pp.Name == "" {
				pp.Name = extractVarName(c.Parent())
			}
			pipelines = append(pipelines, pp)
		}
		return true
	}, nil)

	return pipelines, nil
}

// ParsePipelinesDst extracts all rabbit.NewEventPipeline declarations from a pre-parsed *dst.File.
func ParsePipelinesDst(f *dst.File) []*ParsedPipeline {
	var pipelines []*ParsedPipeline
	dstutil.Apply(f, func(c *dstutil.Cursor) bool {
		call, ok := c.Node().(*dst.CallExpr)
		if !ok {
			return true
		}
		if _, ok := c.Parent().(*dst.SelectorExpr); ok {
			return true
		}
		pp := analyzePipelineChain(call)
		if pp != nil {
			if pp.Name == "" {
				pp.Name = extractVarName(c.Parent())
			}
			pipelines = append(pipelines, pp)
		}
		return true
	}, nil)
	return pipelines
}

// extractVarName extracts the variable name from a dst.Node that is the parent of a
// pipeline CallExpr, handling *dst.ValueSpec (var block) and *dst.AssignStmt (init/func body).
func extractVarName(parent dst.Node) string {
	switch p := parent.(type) {
	case *dst.ValueSpec:
		if len(p.Names) > 0 {
			return p.Names[0].Name
		}
	case *dst.AssignStmt:
		if len(p.Lhs) > 0 {
			if ident, ok := p.Lhs[0].(*dst.Ident); ok {
				return ident.Name
			}
		}
	}
	return ""
}

// analyzePipelineChain recursively descends through chained method calls
// to find the root rabbit.NewEventPipeline[T] call and collect all chained methods.
func analyzePipelineChain(call *dst.CallExpr) *ParsedPipeline {
	// Chained call: .Document(), .MaxRetry(), etc.
	if sel, ok := call.Fun.(*dst.SelectorExpr); ok {
		if innerCall, ok := sel.X.(*dst.CallExpr); ok {
			pp := analyzePipelineChain(innerCall)
			if pp == nil {
				return nil
			}
			collectPipelineChain(pp, sel.Sel.Name, call.Args)
			return pp
		}
	}

	// Base case: rabbit.NewEventPipeline[T](queue)
	if idx, ok := call.Fun.(*dst.IndexExpr); ok {
		if sel, ok := idx.X.(*dst.SelectorExpr); ok {
			if ident, ok := sel.X.(*dst.Ident); ok && ident.Name == "rabbit" && sel.Sel.Name == "NewEventPipeline" {
				pp := &ParsedPipeline{MaxRetry: -1}
				pp.Queue = extractStringArg(call.Args, 0, nil, nil)
				ref := exprToTypeRef(idx.Index)
				pp.PayloadRef = &ref
				return pp
			}
		}
	}

	return nil
}

// collectPipelineChain extracts information from a chained method call on an EventPipeline.
func collectPipelineChain(pp *ParsedPipeline, name string, args []dst.Expr) {
	switch name {
	case "Document":
		pp.Name = extractStringArg(args, 0, nil, nil)
		pp.Desc = extractStringArg(args, 1, nil, nil)
		pp.Provider = extractStringArg(args, 2, nil, nil)
	case "MaxRetry":
		if len(args) > 0 {
			if lit, ok := args[0].(*dst.BasicLit); ok && lit.Kind == token.INT {
				fmt.Sscanf(lit.Value, "%d", &pp.MaxRetry)
			}
		}
	case "LogPayload":
		pp.LogPayload = true
	}
}

// analyzeCallChain recursively descends through chained method calls
// to find the root miso.Http* call and collect all chained methods.
func analyzeCallChain(call *dst.CallExpr, constVars map[string]string, pkg *types.Package) *ParsedEndpoint {
	sel, ok := call.Fun.(*dst.SelectorExpr)
	if !ok {
		return nil
	}

	// If X is another CallExpr, this is a chained method call
	if innerCall, ok := sel.X.(*dst.CallExpr); ok {
		ep := analyzeCallChain(innerCall, constVars, pkg)
		if ep == nil {
			return nil
		}
		collectChained(ep, sel.Sel.Name, call.Args, constVars, pkg)
		return ep
	}

	// Base case: miso.Http*(...)
	if ident, ok := sel.X.(*dst.Ident); ok && ident.Name == "miso" {
		method, ok := httpMethodMap[sel.Sel.Name]
		if !ok {
			return nil
		}
		ep := &ParsedEndpoint{Method: method}
		// URL (first arg)
		if len(call.Args) > 0 {
			ep.URL = wrapUnresolvedURLIdent(call.Args, 0, extractStringArg(call.Args, 0, constVars, pkg))
		}
		// Handler (second arg)
		if len(call.Args) > 1 {
			extractHandler(call.Args[1], ep)
		}
		return ep
	}

	return nil
}

// extractFuncName extracts the FuncName from Extra("miso.ExtraName", ...) if present.
func extractFuncName(ep *ParsedEndpoint) {
	for _, extra := range ep.Extras {
		if extra.Left == "miso.ExtraName" {
			ep.FuncName = extra.Right
			break
		}
	}
}

// processGroupCall handles the miso.BaseRoute("/prefix").Group(endpoints...) pattern.
// Returns endpoints with the base path prepended to each endpoint URL, or nil.
func processGroupCall(call *dst.CallExpr, constVars map[string]string, pkg *types.Package) []*ParsedEndpoint {
	sel, ok := call.Fun.(*dst.SelectorExpr)
	if !ok || sel.Sel.Name != "Group" {
		return nil
	}

	innerCall, ok := sel.X.(*dst.CallExpr)
	if !ok {
		return nil
	}

	innerSel, ok := innerCall.Fun.(*dst.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := innerSel.X.(*dst.Ident)
	if !ok || ident.Name != "miso" || innerSel.Sel.Name != "BaseRoute" {
		return nil
	}

	basePrefix := wrapUnresolvedURLIdent(innerCall.Args, 0, extractStringArg(innerCall.Args, 0, constVars, pkg))
	if basePrefix == "" {
		return nil
	}

	var endpoints []*ParsedEndpoint
	for _, arg := range call.Args {
		argCall, ok := arg.(*dst.CallExpr)
		if !ok {
			continue
		}
		ep := analyzeCallChain(argCall, constVars, pkg)
		if ep != nil {
			ep.URL = basePrefix + ep.URL
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

// collectChained extracts information from a chained method call.
func collectChained(ep *ParsedEndpoint, name string, args []dst.Expr, constVars map[string]string, pkg *types.Package) {
	switch name {
	case "Desc":
		ep.Desc = extractStringArg(args, 0, constVars, pkg)
	case "Public":
		ep.Scope = "PUBLIC"
	case "Protected":
		ep.Scope = "PROTECTED"
	case "Scope":
		ep.Scope = extractStringArg(args, 0, constVars, pkg)
	case "Resource":
		ep.Resource = extractStringArg(args, 0, constVars, pkg)
	case "DocQueryParam":
		ep.QueryParams = append(ep.QueryParams, pair.StrPair{Left: extractStringArg(args, 0, constVars, pkg), Right: extractStringArg(args, 1, constVars, pkg)})
	case "DocHeader":
		ep.Headers = append(ep.Headers, pair.StrPair{Left: extractStringArg(args, 0, constVars, pkg), Right: extractStringArg(args, 1, constVars, pkg)})
	case "DocJsonReq":
		if ep.RequestRef == nil && len(args) > 0 {
			ref := extractTypeRefFromExpr(args[0])
			ep.RequestRef = &ref
		}
	case "DocJsonResp":
		if ep.ResponseRef == nil && len(args) > 0 {
			ref := extractTypeRefFromExpr(args[0])
			ep.ResponseRef = &ref
		}
	case "DocQueryReq":
		if ep.QueryReqType == "" && len(args) > 0 {
			ep.QueryReqType = extractTypeFromExpr(args[0])
		}
	case "DocHeaderReq":
		if ep.HeaderReqType == "" && len(args) > 0 {
			ep.HeaderReqType = extractTypeFromExpr(args[0])
		}
	case "Extra":
		if len(args) >= 2 {
			k := exprToValue(args[0])
			v := exprToValue(args[1])
			ep.Extras = append(ep.Extras, pair.StrPair{Left: k, Right: v})
		}
	}
}

// extractHandler parses the handler argument of miso.Http*.
// Recognizes: miso.AutoHandler(func), miso.ResHandler(func),
// miso.RawHandler(func|ident), and direct function references.
func extractHandler(arg dst.Expr, ep *ParsedEndpoint) {
	switch v := arg.(type) {
	case *dst.CallExpr:
		sel, ok := v.Fun.(*dst.SelectorExpr)
		if !ok {
			return
		}
		if id, ok := sel.X.(*dst.Ident); !ok || id.Name != "miso" {
			return
		}
		ep.Handler = sel.Sel.Name
		if len(v.Args) > 0 {
			if lit, ok := v.Args[0].(*dst.FuncLit); ok {
				extractFuncType(lit.Type, ep)
			}
		}
	case *dst.Ident:
		ep.Handler = "Direct"
	}
}

// extractFuncType pulls request/response types from a function literal's signature.
func extractFuncType(ft *dst.FuncType, ep *ParsedEndpoint) {
	if ft.Params != nil {
		for _, field := range ft.Params.List {
			if isSkipParamType(field.Type) {
				continue
			}
			ref := exprToTypeRef(field.Type)
			ep.RequestRef = &ref
			break
		}
	}
	if ft.Results != nil {
		for _, field := range ft.Results.List {
			if isErrorType(field.Type) {
				continue
			}
			ref := exprToTypeRef(field.Type)
			ep.ResponseRef = &ref
			break
		}
	}
}

// SkipParamType describes a function parameter type that should be excluded from
// the request type extraction (e.g., injectable framework types like *miso.Inbound).
type SkipParamType struct {
	Pkg  string // package name as it appears in the import (e.g., "miso", "gorm")
	Name string // type name without pointer (e.g., "Inbound", "DB")
	Ptr  bool   // whether the param is a pointer type (*pkg.Name)
}

// DefaultSkipParamTypes is the default set of types to skip during parameter type extraction.
// Append to this slice to add custom injectable types without modifying framework code.
var DefaultSkipParamTypes = []SkipParamType{
	{Pkg: "miso", Name: "Inbound", Ptr: true},
	{Pkg: "gorm", Name: "DB", Ptr: true},
	{Pkg: "mysql", Name: "Query", Ptr: true},
	{Pkg: "miso", Name: "Rail", Ptr: false},
	{Pkg: "flow", Name: "User", Ptr: false},
}

// isSkipParamType checks for injectable framework types that should be excluded
// from the request type, using the configurable DefaultSkipParamTypes list.
func isSkipParamType(t dst.Expr) bool {
	for _, st := range DefaultSkipParamTypes {
		if st.Ptr {
			star, ok := t.(*dst.StarExpr)
			if !ok {
				continue
			}
			sel, ok := star.X.(*dst.SelectorExpr)
			if !ok {
				continue
			}
			id, ok := sel.X.(*dst.Ident)
			if ok && id.Name == st.Pkg && sel.Sel.Name == st.Name {
				return true
			}
		} else {
			sel, ok := t.(*dst.SelectorExpr)
			if !ok {
				continue
			}
			id, ok := sel.X.(*dst.Ident)
			if ok && id.Name == st.Pkg && sel.Sel.Name == st.Name {
				return true
			}
		}
	}
	return false
}

// isErrorType checks if a dst.Expr represents the error type.
func isErrorType(t dst.Expr) bool {
	if id, ok := t.(*dst.Ident); ok && id.Name == "error" {
		return true
	}
	return false
}

// collectConstVars walks a *dst.File and collects all const/var declarations
// that have string literal values into a map[name]value.
func collectConstVars(f *dst.File) map[string]string {
	m := map[string]string{}
	collectConstVarsInto(f, m)
	return m
}

// CollectPackageConstVars collects const/var string values from multiple files
// into a merged map. File-local values take precedence (first file wins if duplicate).
func CollectPackageConstVars(files []*dst.File) map[string]string {
	m := map[string]string{}
	for _, f := range files {
		collectConstVarsInto(f, m)
	}
	return m
}

func collectConstVarsInto(f *dst.File, m map[string]string) {
	dstutil.Apply(f, func(c *dstutil.Cursor) bool {
		n, ok := c.Node().(*dst.GenDecl)
		if !ok || (n.Tok != token.CONST && n.Tok != token.VAR) {
			return true
		}
		for _, spec := range n.Specs {
			vs, ok := spec.(*dst.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				if lit, ok := vs.Values[i].(*dst.BasicLit); ok && lit.Kind == token.STRING {
					s := lit.Value
					if len(s) >= 2 {
						m[name.Name] = s[1 : len(s)-1]
					}
				}
			}
		}
		return true
	}, nil)
}

// resolveConstStr resolves a types.Object to its string value if it's a string constant.
func resolveConstStr(obj types.Object) string {
	if obj == nil {
		return ""
	}
	c, ok := obj.(*types.Const)
	if !ok {
		return ""
	}
	if c.Val() == nil || c.Val().Kind() != constant.String {
		return ""
	}
	return constant.StringVal(c.Val())
}

// resolveImportedConstStr resolves a pkgName.ConstName reference to its string value
// by looking up the const in the imported package's scope.
func resolveImportedConstStr(pkg *types.Package, pkgName, constName string) string {
	for _, imp := range pkg.Imports() {
		if imp.Name() == pkgName {
			if obj := imp.Scope().Lookup(constName); obj != nil {
				if c, ok := obj.(*types.Const); ok {
					if c.Val() != nil && c.Val().Kind() == constant.String {
						return constant.StringVal(c.Val())
					}
				}
			}
			break
		}
	}
	return ""
}

// wrapUnresolvedURLIdent wraps the resolved string in /${...} if it came from an
// unresolved *dst.Ident (a variable that couldn't be resolved to a const value).
// Only use for URL paths; do NOT use for Desc/Scope/Resource etc.
func wrapUnresolvedURLIdent(args []dst.Expr, idx int, resolved string) string {
	if ident, ok := args[idx].(*dst.Ident); ok && resolved == ident.Name {
		return "/${" + resolved + "}"
	}
	return resolved
}

// extractStringArg extracts a string literal from args[idx].
// For non-literal expressions, first tries to resolve as a const/var reference,
// then falls back to exprToString.
func extractStringArg(args []dst.Expr, idx int, constVars map[string]string, pkg *types.Package) string {
	if idx >= len(args) {
		return ""
	}
	if lit, ok := args[idx].(*dst.BasicLit); ok && lit.Kind == token.STRING {
		s := lit.Value
		if len(s) >= 2 {
			return s[1 : len(s)-1]
		}
	}
	// Try to resolve as const/var reference (AST-level)
	if ident, ok := args[idx].(*dst.Ident); ok {
		if constVars != nil {
			if val, ok := constVars[ident.Name]; ok {
				return val
			}
		}
		// Fallback: try package scope (local consts/var)
		if pkg != nil {
			if val := resolveConstStr(pkg.Scope().Lookup(ident.Name)); val != "" {
				return val
			}
		}
	}
	// Handle SelectorExpr: pkgName.ConstName (imported consts)
	if sel, ok := args[idx].(*dst.SelectorExpr); ok && pkg != nil {
		if pkgIdent, ok := sel.X.(*dst.Ident); ok {
			if val := resolveImportedConstStr(pkg, pkgIdent.Name, sel.Sel.Name); val != "" {
				return val
			}
		}
	}
	// Handle BinaryExpr: string concatenation (e.g., "prefix" + constName)
	if bin, ok := args[idx].(*dst.BinaryExpr); ok && bin.Op == token.ADD {
		left := extractStringArg([]dst.Expr{bin.X}, 0, constVars, pkg)
		right := extractStringArg([]dst.Expr{bin.Y}, 0, constVars, pkg)
		return left + right
	}
	// Fallback: capture variable identifiers (e.g., deregisterURL)
	return exprToString(args[idx])
}

// extractTypeFromExpr extracts a type name from an expression.
// Handles CompositeLit (e.g., ApiReq{}) and plain identifiers.
func extractTypeFromExpr(e dst.Expr) string {
	switch v := e.(type) {
	case *dst.CompositeLit:
		return exprToString(v.Type)
	case *dst.Ident:
		return v.Name
	case *dst.SelectorExpr:
		return exprToString(v)
	}
	return ""
}

// exprToValue converts a dst.Expr to its string representation for Extra key/value.
func exprToValue(e dst.Expr) string {
	switch v := e.(type) {
	case *dst.BasicLit:
		if v.Kind == token.STRING {
			s := v.Value
			if len(s) >= 2 {
				return s[1 : len(s)-1]
			}
		}
		return v.Value
	case *dst.Ident:
		return v.Name
	case *dst.SelectorExpr:
		return exprToString(v)
	default:
		return fmt.Sprintf("?%T", e)
	}
}

// exprToString converts a type expression (dst.Expr) to its string representation.
// Handles: *dst.Ident, *dst.StarExpr, *dst.SelectorExpr, *dst.ArrayType,
// *dst.IndexExpr (generics), *dst.MapType.
func exprToString(t dst.Expr) string {
	switch v := t.(type) {
	case *dst.Ident:
		return v.Name
	case *dst.SelectorExpr:
		return exprToString(v.X) + "." + v.Sel.Name
	case *dst.StarExpr:
		return "*" + exprToString(v.X)
	case *dst.ArrayType:
		return "[]" + exprToString(v.Elt)
	case *dst.IndexExpr:
		return exprToString(v.X) + "[" + exprToString(v.Index) + "]"
	case *dst.MapType:
		return "map[" + exprToString(v.Key) + "]" + exprToString(v.Value)
	default:
		return fmt.Sprintf("?%T", t)
	}
}

// exprToTypeRef converts a dst.Expr (type expression) to a structured TypeRef.
// This preserves type wrappers (pointer, slice, map) and generic type arguments
// that are lost in the string round-trip of exprToString → lookupAndBuildTypeDesc.
func exprToTypeRef(t dst.Expr) TypeRef {
	switch v := t.(type) {
	case *dst.Ident:
		if isBuiltIn(v.Name) {
			return TypeRef{Name: v.Name}
		}
		return TypeRef{Name: v.Name}

	case *dst.SelectorExpr:
		if left, ok := v.X.(*dst.Ident); ok {
			// pkg.Type pattern — keep PkgName and Name separate
			return TypeRef{PkgName: left.Name, Name: v.Sel.Name}
		}
		inner := exprToTypeRef(v.X)
		inner.Name = inner.Name + "." + v.Sel.Name
		return inner

	case *dst.StarExpr:
		inner := exprToTypeRef(v.X)
		inner.IsPtr = true
		return inner

	case *dst.ArrayType:
		inner := exprToTypeRef(v.Elt)
		if _, ok := v.Elt.(*dst.StarExpr); ok {
			// []*T → IsSlicePtr flags the * inside [], IsSlice flags the outer []
			inner.IsSlice = true
			inner.IsSlicePtr = inner.IsPtr
			inner.IsPtr = false
			return inner
		}
		// plain []T
		inner.IsSlice = true
		return inner

	case *dst.IndexExpr:
		base := exprToTypeRef(v.X)
		base.TypeArgs = append(base.TypeArgs, exprToTypeRef(v.Index))
		return base

	case *dst.MapType:
		key := exprToTypeRef(v.Key)
		val := exprToTypeRef(v.Value)
		return TypeRef{
			IsMap:    true,
			MapKey:   &key,
			MapValue: &val,
		}

	default:
		return TypeRef{Name: fmt.Sprintf("?%T", t)}
	}
}

// extractTypeRefFromExpr extracts a TypeRef from an expression that contains
// a type, handling CompositeLit (e.g., ApiReq{}) and plain idents.
func extractTypeRefFromExpr(e dst.Expr) TypeRef {
	switch v := e.(type) {
	case *dst.CompositeLit:
		return exprToTypeRef(v.Type)
	case *dst.Ident:
		return TypeRef{Name: v.Name}
	case *dst.SelectorExpr:
		return exprToTypeRef(v)
	}
	return TypeRef{}
}

// isBuiltIn reports whether name is a Go built-in type.
func isBuiltIn(name string) bool {
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
