package sourceparser

import (
	"fmt"
	"go/parser"
	"go/token"
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
		if eps := processGroupCall(call); len(eps) > 0 {
			for _, ep := range eps {
				extractFuncName(ep)
				endpoints = append(endpoints, ep)
			}
			return false // Don't recurse into Group's children (already processed)
		}

		ep := analyzeCallChain(call)
		if ep != nil {
			extractFuncName(ep)
			endpoints = append(endpoints, ep)
		}
		return true
	}, nil)

	return endpoints, nil
}

// analyzeCallChain recursively descends through chained method calls
// to find the root miso.Http* call and collect all chained methods.
func analyzeCallChain(call *dst.CallExpr) *ParsedEndpoint {
	sel, ok := call.Fun.(*dst.SelectorExpr)
	if !ok {
		return nil
	}

	// If X is another CallExpr, this is a chained method call
	if innerCall, ok := sel.X.(*dst.CallExpr); ok {
		ep := analyzeCallChain(innerCall)
		if ep == nil {
			return nil
		}
		collectChained(ep, sel.Sel.Name, call.Args)
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
			ep.URL = extractStringArg(call.Args, 0)
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
func processGroupCall(call *dst.CallExpr) []*ParsedEndpoint {
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

	basePrefix := extractStringArg(innerCall.Args, 0)
	if basePrefix == "" {
		return nil
	}

	var endpoints []*ParsedEndpoint
	for _, arg := range call.Args {
		argCall, ok := arg.(*dst.CallExpr)
		if !ok {
			continue
		}
		ep := analyzeCallChain(argCall)
		if ep != nil {
			ep.URL = basePrefix + ep.URL
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints
}

// collectChained extracts information from a chained method call.
func collectChained(ep *ParsedEndpoint, name string, args []dst.Expr) {
	switch name {
	case "Desc":
		ep.Desc = extractStringArg(args, 0)
	case "Public":
		ep.Scope = "PUBLIC"
	case "Protected":
		ep.Scope = "PROTECTED"
	case "Scope":
		ep.Scope = extractStringArg(args, 0)
	case "Resource":
		ep.Resource = extractStringArg(args, 0)
	case "DocQueryParam":
		ep.QueryParams = append(ep.QueryParams, pair.StrPair{Left: extractStringArg(args, 0), Right: extractStringArg(args, 1)})
	case "DocHeader":
		ep.Headers = append(ep.Headers, pair.StrPair{Left: extractStringArg(args, 0), Right: extractStringArg(args, 1)})
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

// extractStringArg extracts a string literal from args[idx].
// Falls back to exprToString for non-literal expressions (e.g., variable URLs).
func extractStringArg(args []dst.Expr, idx int) string {
	if idx >= len(args) {
		return ""
	}
	if lit, ok := args[idx].(*dst.BasicLit); ok && lit.Kind == token.STRING {
		s := lit.Value
		if len(s) >= 2 {
			return s[1 : len(s)-1]
		}
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
