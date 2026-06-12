package docgen

import (
	"fmt"
	"go/constant"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/miso/sourceparser"
	"github.com/curtisnewbie/miso/util/pair"
	"golang.org/x/tools/go/packages"
)

// === resolveTypeAlias unit tests ===

func TestResolveTypeAlias(t *testing.T) {
	// resolveTypeAlias(rawOrigin, pkgOrigin, pureOrigin) → (alias, matched)
	//
	// rawOrigin:  types.TypeString(t, qual) — full module path
	// pkgOrigin:  types.TypeString(t, pkgNameQual) — short pkg name
	// pureOrigin: miso.PureGoTypeName(rawOrigin) — bare type name
	tests := []struct {
		name       string
		rawOrigin  string
		pkgOrigin  string
		pureOrigin string
		wantAlias  string
		wantMatch  bool
	}{
		// atom.Time → int64
		{
			name: "atom.Time full path", rawOrigin: "github.com/curtisnewbie/miso/util/atom.Time",
			pkgOrigin: "atom.Time", pureOrigin: "Time",
			wantAlias: "int64", wantMatch: true,
		},
		{
			name: "*atom.Time full path", rawOrigin: "*github.com/curtisnewbie/miso/util/atom.Time",
			pkgOrigin: "*atom.Time", pureOrigin: "Time",
			wantAlias: "int64", wantMatch: true,
		},
		// Note: types.TypeString for types in a different package always produces
		// the full module-qualified path. Unqualified names like "Time" only occur
		// for types in the same package, and ApiDocTypeAlias only has external types.
		// So there is no test case for unqualified rawOrigin matching aliases.

		// money.Amt → string
		{
			name: "money.Amt full path", rawOrigin: "github.com/curtisnewbie/miso/middleware/money.Amt",
			pkgOrigin: "money.Amt", pureOrigin: "Amt",
			wantAlias: "string", wantMatch: true,
		},
		{
			name: "*money.Amt full path", rawOrigin: "*github.com/curtisnewbie/miso/middleware/money.Amt",
			pkgOrigin: "*money.Amt", pureOrigin: "Amt",
			wantAlias: "string", wantMatch: true,
		},
		// See note above: unqualified names don't match aliases in practice.

		// hash.Set[string] → []string
		{
			name: "hash.Set[string] full path", rawOrigin: "github.com/curtisnewbie/miso/util/hash.Set[string]",
			pkgOrigin: "hash.Set[string]", pureOrigin: "Set",
			wantAlias: "[]string", wantMatch: true,
		},
		{
			name: "*hash.Set[string] full path", rawOrigin: "*github.com/curtisnewbie/miso/util/hash.Set[string]",
			pkgOrigin: "*hash.Set[string]", pureOrigin: "Set",
			wantAlias: "[]string", wantMatch: true,
		},

		// hash.Set[int64] → []int64
		{
			name: "hash.Set[int64] full path", rawOrigin: "github.com/curtisnewbie/miso/util/hash.Set[int64]",
			pkgOrigin: "hash.Set[int64]", pureOrigin: "Set",
			wantAlias: "[]int64", wantMatch: true,
		},

		// Unaliased types — fallback behavior
		{
			name: "string builtin", rawOrigin: "string",
			pkgOrigin: "string", pureOrigin: "string",
			wantAlias: "string", wantMatch: false,
		},
		{
			name: "*int pointer builtin", rawOrigin: "*int",
			pkgOrigin: "*int", pureOrigin: "int",
			wantAlias: "*int", wantMatch: false, // falls back to pkgOrigin (has * prefix)
		},
		{
			name: "[]string slice", rawOrigin: "[]string",
			pkgOrigin: "[]string", pureOrigin: "string",
			wantAlias: "[]string", wantMatch: false, // falls back to pkgOrigin (has [] prefix)
		},
		{
			name: "unaliased custom type", rawOrigin: "github.com/example/foo.Bar",
			pkgOrigin: "foo.Bar", pureOrigin: "Bar",
			wantAlias: "Bar", wantMatch: false, // falls back to pureOrigin (no ptr/slice prefix)
		},
		{
			name: "*unaliased custom type", rawOrigin: "*github.com/example/foo.Bar",
			pkgOrigin: "*foo.Bar", pureOrigin: "Bar",
			wantAlias: "*foo.Bar", wantMatch: false, // falls back to pkgOrigin (has * prefix)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAlias, gotMatch := resolveTypeAlias(tt.rawOrigin, tt.pkgOrigin, tt.pureOrigin)
			if gotAlias != tt.wantAlias {
				t.Errorf("alias = %q, want %q", gotAlias, tt.wantAlias)
			}
			if gotMatch != tt.wantMatch {
				t.Errorf("matched = %v, want %v", gotMatch, tt.wantMatch)
			}
		})
	}
}

// === buildDataField unit tests ===

func TestBuildDataField(t *testing.T) {
	tests := []struct {
		name           string
		dto            miso.TypeDesc
		wantName       string // expected TypeNameAlias
		wantIsPtr      bool
		wantIsSlice    bool
		wantIsSlicePtr bool
		wantIsMap      bool
	}{
		{
			name:     "plain value type",
			dto:      miso.TypeDesc{TypeName: "PostRes", TypePkg: "github.com/curtisnewbie/miso/demo/api"},
			wantName: "PostRes",
		},
		{
			name:      "pointer type gets pkg prefix",
			dto:       miso.TypeDesc{TypeName: "PostRes", TypePkg: "github.com/curtisnewbie/miso/demo/api", IsPtr: true},
			wantName:  "*api.PostRes",
			wantIsPtr: true,
		},
		{
			name:        "slice type gets pkg prefix",
			dto:         miso.TypeDesc{TypeName: "PostRes", TypePkg: "github.com/curtisnewbie/miso/demo/api", IsSlice: true},
			wantName:    "[]api.PostRes",
			wantIsSlice: true,
		},
		{
			name:           "slice of pointer type",
			dto:            miso.TypeDesc{TypeName: "ApiRes", TypePkg: "github.com/curtisnewbie/miso/demo/api", IsSlice: true, IsSlicePtr: true},
			wantName:       "[]*api.ApiRes",
			wantIsSlice:    true,
			wantIsSlicePtr: true,
		},
		{
			name:        "pointer to slice",
			dto:         miso.TypeDesc{TypeName: "PostRes", TypePkg: "github.com/curtisnewbie/miso/demo/api", IsPtr: true, IsSlice: true},
			wantName:    "*[]api.PostRes",
			wantIsPtr:   true,
			wantIsSlice: true,
		},
		{
			name:     "value type already has pkg prefix in name",
			dto:      miso.TypeDesc{TypeName: "api.PostRes", TypePkg: "github.com/curtisnewbie/miso/demo/api"},
			wantName: "api.PostRes",
		},
		{
			name:      "pointer type already has pkg prefix in name",
			dto:       miso.TypeDesc{TypeName: "api.PostRes", TypePkg: "github.com/curtisnewbie/miso/demo/api", IsPtr: true},
			wantName:  "*api.PostRes",
			wantIsPtr: true,
		},
		{
			name:      "map type",
			dto:       miso.TypeDesc{TypeName: "map[string]int32", TypePkg: ""},
			wantName:  "map[string]int32",
			wantIsMap: true,
		},
		{
			name:     "builtin string type",
			dto:      miso.TypeDesc{TypeName: "string", TypePkg: ""},
			wantName: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := buildDataField(tt.dto)
			if fd.GoFieldName != "Data" {
				t.Errorf("GoFieldName = %q, want %q", fd.GoFieldName, "Data")
			}
			if fd.JsonName != "data" {
				t.Errorf("JsonName = %q, want %q", fd.JsonName, "data")
			}
			if fd.TypeNameAlias != tt.wantName {
				t.Errorf("TypeNameAlias = %q, want %q", fd.TypeNameAlias, tt.wantName)
			}
			if fd.IsPointer != tt.wantIsPtr {
				t.Errorf("IsPointer = %v, want %v", fd.IsPointer, tt.wantIsPtr)
			}
			if fd.IsSliceOrArray != tt.wantIsSlice {
				t.Errorf("IsSliceOrArray = %v, want %v", fd.IsSliceOrArray, tt.wantIsSlice)
			}
			if fd.IsSliceOfPointer != tt.wantIsSlicePtr {
				t.Errorf("IsSliceOfPointer = %v, want %v", fd.IsSliceOfPointer, tt.wantIsSlicePtr)
			}
			if fd.IsMap != tt.wantIsMap {
				t.Errorf("IsMap = %v, want %v", fd.IsMap, tt.wantIsMap)
			}
		})
	}
}

// === hasExtra unit tests ===

func TestHasExtra(t *testing.T) {
	tests := []struct {
		name       string
		extras     []pair.StrPair
		extraConst string
		want       bool
	}{
		{
			name:       "ExtraNgTable matches",
			extras:     []pair.StrPair{{Left: "miso.ExtraNgTable", Right: "true"}},
			extraConst: miso.ExtraNgTable,
			want:       true,
		},
		{
			name:       "ExtraName matches",
			extras:     []pair.StrPair{{Left: "miso.ExtraName", Right: "api1"}},
			extraConst: miso.ExtraName,
			want:       true,
		},
		{
			name:       "wrong const does not match",
			extras:     []pair.StrPair{{Left: "miso.ExtraName", Right: "api1"}},
			extraConst: miso.ExtraNgTable,
			want:       false,
		},
		{
			name:       "empty extras",
			extras:     nil,
			extraConst: miso.ExtraNgTable,
			want:       false,
		},
		{
			name:       "non-miso extras ignored",
			extras:     []pair.StrPair{{Left: "custom.ExtraFoo", Right: "bar"}},
			extraConst: miso.ExtraName,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasExtra(tt.extras, tt.extraConst)
			if got != tt.want {
				t.Errorf("hasExtra() = %v, want %v", got, tt.want)
			}
		})
	}
}

// === Integration test: BuildManualRouteDocs on demo/appdemo ===

// findDoc returns the HttpRouteDoc matching method and url, or nil.
func findDoc(docs []miso.HttpRouteDoc, method, url string) *miso.HttpRouteDoc {
	for i := range docs {
		if docs[i].Method == method && docs[i].Url == url {
			return &docs[i]
		}
	}
	return nil
}

// findField recursively finds a field by json name.
func findField(fields []miso.FieldDesc, jsonName string) *miso.FieldDesc {
	for i := range fields {
		if fields[i].JsonName == jsonName {
			return &fields[i]
		}
		if f := findField(fields[i].Fields, jsonName); f != nil {
			return f
		}
	}
	return nil
}

// TestBuildManualRouteDocs_DemoAppdemo runs the full doc generation pipeline
// against demo/appdemo source files and verifies edge cases from HANDOFF.md.
func TestBuildManualRouteDocs_DemoAppdemo(t *testing.T) {
	// Find the demo/appdemo directory relative to the test file.
	// miso/docgen/ → ../../demo/appdemo
	demoDir, err := filepath.Abs(filepath.Join("..", "..", "demo", "appdemo"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if _, err := os.Stat(demoDir); os.IsNotExist(err) {
		t.Skipf("demo/appdemo directory not found at %s", demoDir)
	}

	// Ensure misoapi_generated.go exists
	if _, err := os.Stat(filepath.Join(demoDir, "api", "misoapi_generated.go")); os.IsNotExist(err) {
		t.Skip("misoapi_generated.go not found")
	}

	// Chdir to demo/appdemo so SourceFile paths are relative to the module root
	// (BuildManualRouteDocs constructs pkgPath as modName + "/" + dir).
	origDir, _ := os.Getwd()
	if err := os.Chdir(demoDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer os.Chdir(origDir)

	// Collect only misoapi_generated.go to avoid duplicates (api.go has the
	// same endpoint definitions via comments, which sourceparser also picks up).
	files := []SourceFile{{Path: "api/misoapi_generated.go"}}
	modName := "github.com/curtisnewbie/miso/demo"

	docs := BuildManualRouteDocs(files, modName, nopLogger{}, nil, nil)

	// Should find at least the auto-generated endpoints (33)
	if len(docs) < 30 {
		t.Errorf("expected at least 30 endpoints, got %d", len(docs))
	}
	t.Logf("Found %d endpoints", len(docs))

	// === Edge case: POST /api/v1 — Resp wrapper with Data: (PostRes) ===
	t.Run("POST /api/v1 Resp wrapper", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v1")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		resp := doc.JsonResponseDesc
		if resp.TypeName != "Resp" {
			t.Errorf("response TypeName = %q, want %q", resp.TypeName, "Resp")
		}
		// Resp must have errorCode, msg, error, data fields
		if f := findField(resp.Fields, "errorCode"); f == nil {
			t.Error("errorCode field missing from Resp")
		}
		if f := findField(resp.Fields, "msg"); f == nil {
			t.Error("msg field missing from Resp")
		}
		if f := findField(resp.Fields, "error"); f == nil {
			t.Error("error field missing from Resp")
		}
		dataField := findField(resp.Fields, "data")
		if dataField == nil {
			t.Fatal("data field missing from Resp")
		}
		if dataField.GoFieldName != "Data" {
			t.Errorf("data GoFieldName = %q, want %q", dataField.GoFieldName, "Data")
		}
		if !strings.Contains(dataField.TypeNameAlias, "PostRes") {
			t.Errorf("data TypeNameAlias = %q, want to contain %q", dataField.TypeNameAlias, "PostRes")
		}
	})

	// === Edge case: POST /api/v3 — pointer with pkg prefix ===
	t.Run("POST /api/v3 pointer with pkg prefix", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v3")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		dataField := findField(doc.JsonResponseDesc.Fields, "data")
		if dataField == nil {
			t.Fatal("data field missing")
		}
		if dataField.TypeNameAlias != "*api.PostRes" {
			t.Errorf("data TypeNameAlias = %q, want %q", dataField.TypeNameAlias, "*api.PostRes")
		}
	})

	// === Edge case: POST /api/v9 — slice-of-pointer ([]*api.ApiRes) ===
	t.Run("POST /api/v9 slice-of-pointer", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v9")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		dataField := findField(doc.JsonResponseDesc.Fields, "data")
		if dataField == nil {
			t.Fatal("data field missing")
		}
		if dataField.TypeNameAlias != "[]*api.ApiRes" {
			t.Errorf("data TypeNameAlias = %q, want %q", dataField.TypeNameAlias, "[]*api.ApiRes")
		}
		if !dataField.IsSliceOrArray {
			t.Error("IsSliceOrArray should be true")
		}
		if !dataField.IsSliceOfPointer {
			t.Error("IsSliceOfPointer should be true")
		}
	})

	// === Edge case: GET /api/v16 — PageRes[PostRes], NgTable demo ===
	t.Run("GET /api/v16 PageRes and NgTable", func(t *testing.T) {
		doc := findDoc(docs, "GET", "/api/v16")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		// Should have NgTable demo
		if doc.NgTableDemo == "" {
			t.Error("NgTableDemo should not be empty")
		}
		if !strings.Contains(doc.NgTableDemo, "mat-table") {
			t.Error("NgTableDemo should contain mat-table markup")
		}
		// Response should be Resp wrapping PageRes[PostRes]
		resp := doc.JsonResponseDesc
		if resp.TypeName != "Resp" {
			t.Errorf("response TypeName = %q, want %q", resp.TypeName, "Resp")
		}
		dataField := findField(resp.Fields, "data")
		if dataField == nil {
			t.Fatal("data field missing")
		}
		if !strings.Contains(dataField.TypeNameAlias, "PageRes") {
			t.Errorf("data TypeNameAlias = %q, want to contain %q", dataField.TypeNameAlias, "PageRes")
		}
		// NgTable demo should contain date pipe for atom.Time fields (TypeNameAlias == "int64")
		if !strings.Contains(doc.NgTableDemo, "date:") {
			t.Error("NgTableDemo should contain date pipe for time fields")
		}
	})

	// === Edge case: POST /api/v18 — Resp without Data (error-only handler) ===
	t.Run("POST /api/v18 Resp without Data", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v18")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		resp := doc.JsonResponseDesc
		if resp.TypeName != "Resp" {
			t.Errorf("response TypeName = %q, want %q", resp.TypeName, "Resp")
		}
		// Must have errorCode, msg, error
		if findField(resp.Fields, "errorCode") == nil {
			t.Error("errorCode missing")
		}
		if findField(resp.Fields, "msg") == nil {
			t.Error("msg missing")
		}
		if findField(resp.Fields, "error") == nil {
			t.Error("error missing")
		}
		// Must NOT have data field
		if f := findField(resp.Fields, "data"); f != nil {
			t.Error("data field should be absent for any-response handlers")
		}
	})

	// === Edge case: POST /api/v21 — RawHandler with DocJsonReq ===
	t.Run("POST /api/v21 RawHandler DocJsonReq", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v21")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		req := doc.JsonRequestDesc
		if len(req.Fields) == 0 {
			t.Error("request should have fields from ApiReq")
		}
		// ApiReq has Name and Extras fields
		if f := findField(req.Fields, "name"); f == nil {
			t.Error("name field missing from request")
		}
	})

	// === Edge case: POST /api/v22 — RawHandler with DocJsonResp, no Resp wrapper ===
	t.Run("POST /api/v22 RawHandler DocJsonResp no Resp wrapper", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v22")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		resp := doc.JsonResponseDesc
		// RawHandler + DocJsonResp → raw DTO, NOT wrapped in Resp
		if resp.TypeName == "Resp" {
			t.Errorf("response should not be wrapped in Resp, got TypeName = %q", resp.TypeName)
		}
		if resp.TypeName == "" {
			t.Error("response TypeName should not be empty")
		}
		// PostRes has resultId and time fields
		if f := findField(resp.Fields, "resultId"); f == nil {
			t.Error("resultId field missing from response")
		}
	})

	// === Edge case: POST /api/v33 — type alias resolution (amt→string, set→[]string) ===
	t.Run("POST /api/v33 type aliases", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v33")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		// AutoHandler → Resp wrapping → Data contains PostRes2 fields
		dataField := findField(doc.JsonResponseDesc.Fields, "data")
		if dataField == nil {
			t.Fatal("data field missing")
		}

		// amt field: money.Amt → "string"
		amtField := findField(dataField.Fields, "amt")
		if amtField == nil {
			t.Fatal("amt field missing from response data")
		}
		if amtField.TypeNameAlias != "string" {
			t.Errorf("amt TypeNameAlias = %q, want %q", amtField.TypeNameAlias, "string")
		}

		// amtPtr field: *money.Amt → "*string" (ptr + aliased string)
		amtPtrField := findField(dataField.Fields, "amtPtr")
		if amtPtrField == nil {
			t.Fatal("amtPtr field missing from response data")
		}
		if amtPtrField.TypeNameAlias != "*string" {
			t.Errorf("amtPtr TypeNameAlias = %q, want %q", amtPtrField.TypeNameAlias, "*string")
		}

		// set field: hash.Set[string] → "[]string"
		// Note: IsSliceOrArray is NOT set because hash.Set[string] is a named
		// Go struct (not a slice), even though its alias display is "[]string".
		// Both static and runtime code share this behavior.
		setField := findField(dataField.Fields, "set")
		if setField == nil {
			t.Fatal("set field missing from response data")
		}
		if setField.TypeNameAlias != "[]string" {
			t.Errorf("set TypeNameAlias = %q, want %q", setField.TypeNameAlias, "[]string")
		}

		// time field: atom.Time → "int64"
		timeField := findField(dataField.Fields, "time")
		if timeField == nil {
			t.Fatal("time field missing from response data")
		}
		if timeField.TypeNameAlias != "int64" {
			t.Errorf("time TypeNameAlias = %q, want %q", timeField.TypeNameAlias, "int64")
		}
	})

	// === Edge case: POST /api/v32 — map response type ===
	t.Run("POST /api/v32 map response", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v32")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		dataField := findField(doc.JsonResponseDesc.Fields, "data")
		if dataField == nil {
			t.Fatal("data field missing")
		}
		if !dataField.IsMap {
			t.Error("data IsMap should be true for map response")
		}
		if !strings.HasPrefix(dataField.TypeNameAlias, "map[") {
			t.Errorf("data TypeNameAlias = %q, want map[...]", dataField.TypeNameAlias)
		}
	})

	// === Edge case: POST /api/v10 — []ApiRes (slice of value type) ===
	t.Run("POST /api/v10 slice of values", func(t *testing.T) {
		doc := findDoc(docs, "POST", "/api/v10")
		if doc == nil {
			t.Fatal("endpoint not found")
		}
		dataField := findField(doc.JsonResponseDesc.Fields, "data")
		if dataField == nil {
			t.Fatal("data field missing")
		}
		if !dataField.IsSliceOrArray {
			t.Error("IsSliceOrArray should be true")
		}
		if dataField.IsSliceOfPointer {
			t.Error("IsSliceOfPointer should be false (slice of values, not pointers)")
		}
		if !strings.Contains(dataField.TypeNameAlias, "[]") {
			t.Errorf("TypeNameAlias = %q, want [] prefix", dataField.TypeNameAlias)
		}
	})
}

// === BuildTypeDescFromType direct unit tests ===

// loadDemoAPIPkg loads the demo/appdemo/api package for type-level tests.
// Uses go/packages with absolute paths so it works regardless of CWD.
var (
	demoAPIPkgOnce sync.Once
	demoAPIPkg     *types.Package
	demoAPIPkgErr  error
)

func loadDemoAPIPkg(t *testing.T) *types.Package {
	t.Helper()
	demoAPIPkgOnce.Do(func() {
		demoDir, err := filepath.Abs(filepath.Join("..", "..", "demo", "appdemo"))
		if err != nil {
			demoAPIPkgErr = fmt.Errorf("filepath.Abs: %w", err)
			return
		}
		cfg := &packages.Config{
			Mode: packages.NeedTypes | packages.NeedName | packages.NeedImports | packages.NeedDeps,
			Dir:  filepath.Join(demoDir, "api"),
		}
		pkgs, loadErr := packages.Load(cfg, "github.com/curtisnewbie/miso/demo/api")
		if loadErr != nil || len(pkgs) == 0 || pkgs[0].Types == nil {
			demoAPIPkgErr = fmt.Errorf("packages.Load: %w", loadErr)
			return
		}
		demoAPIPkg = pkgs[0].Types
	})
	if demoAPIPkgErr != nil {
		t.Fatalf("loadDemoAPIPkg: %v", demoAPIPkgErr)
	}
	return demoAPIPkg
}

func TestBuildTypeDescFromType_Struct(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	obj := pkg.Scope().Lookup("PostRes")
	if obj == nil {
		t.Fatal("PostRes not found")
	}
	desc := BuildTypeDescFromType(obj.Type(), pkg)
	if desc.TypeName != "PostRes" {
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "PostRes")
	}
	if desc.TypePkg != "github.com/curtisnewbie/miso/demo/api" {
		t.Errorf("TypePkg = %q, want %q", desc.TypePkg, "github.com/curtisnewbie/miso/demo/api")
	}
	if desc.IsPtr || desc.IsSlice || desc.IsSlicePtr {
		t.Error("PostRes should not have pointer/slice flags")
	}
	if len(desc.Fields) != 2 {
		t.Fatalf("Fields len = %d, want 2 (resultId, time)", len(desc.Fields))
	}
	// resultId field
	if desc.Fields[0].JsonName != "resultId" {
		t.Errorf("Fields[0].JsonName = %q, want %q", desc.Fields[0].JsonName, "resultId")
	}
	if desc.Fields[0].TypeNameAlias != "string" {
		t.Errorf("Fields[0].TypeNameAlias = %q, want %q", desc.Fields[0].TypeNameAlias, "string")
	}
	// time field (atom.Time → "int64" alias)
	if desc.Fields[1].JsonName != "time" {
		t.Errorf("Fields[1].JsonName = %q, want %q", desc.Fields[1].JsonName, "time")
	}
	if desc.Fields[1].TypeNameAlias != "int64" {
		t.Errorf("Fields[1].TypeNameAlias = %q, want %q", desc.Fields[1].TypeNameAlias, "int64")
	}
}

func TestBuildTypeDescFromType_Pointer(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	obj := pkg.Scope().Lookup("PostRes")
	ptrType := types.NewPointer(obj.Type())
	desc := BuildTypeDescFromType(ptrType, pkg)
	if !desc.IsPtr {
		t.Error("IsPtr should be true for *PostRes")
	}
	if desc.TypeName != "PostRes" {
		// TypeName comes from the named type, not the pointer wrapper
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "PostRes")
	}
	if len(desc.Fields) != 2 {
		t.Errorf("pointer to struct should preserve fields, got %d", len(desc.Fields))
	}
}

func TestBuildTypeDescFromType_Slice(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	obj := pkg.Scope().Lookup("PostRes")
	sliceType := types.NewSlice(obj.Type())
	desc := BuildTypeDescFromType(sliceType, pkg)
	if !desc.IsSlice {
		t.Error("IsSlice should be true for []PostRes")
	}
	if desc.IsSlicePtr {
		t.Error("IsSlicePtr should be false for []PostRes (not []*PostRes)")
	}
	if desc.TypeName != "PostRes" {
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "PostRes")
	}
	if len(desc.Fields) != 2 {
		t.Errorf("slice of struct should preserve fields, got %d", len(desc.Fields))
	}
}

func TestBuildTypeDescFromType_SliceOfPointer(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	obj := pkg.Scope().Lookup("PostRes")
	ptrType := types.NewPointer(obj.Type())
	slicePtrType := types.NewSlice(ptrType)
	desc := BuildTypeDescFromType(slicePtrType, pkg)
	if !desc.IsSlice {
		t.Error("IsSlice should be true for []*PostRes")
	}
	if !desc.IsSlicePtr {
		t.Error("IsSlicePtr should be true for []*PostRes")
	}
	if desc.IsPtr {
		t.Error("IsPtr should be false (pointer is inside slice)")
	}
	if len(desc.Fields) != 2 {
		t.Errorf("slice of pointers to struct should preserve fields, got %d", len(desc.Fields))
	}
}

func TestBuildTypeDescFromType_Map(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	mapType := types.NewMap(
		types.Universe.Lookup("string").Type(),
		types.Universe.Lookup("int32").Type(),
	)
	desc := BuildTypeDescFromType(mapType, pkg)
	if desc.TypeName == "" {
		t.Error("map type should have a TypeName")
	}
	if !strings.HasPrefix(desc.TypeName, "map[") {
		t.Errorf("map TypeName = %q, want map[...]", desc.TypeName)
	}
}

func TestBuildTypeDescFromType_NestedStruct(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	obj := pkg.Scope().Lookup("ApiReq")
	if obj == nil {
		t.Fatal("ApiReq not found")
	}
	desc := BuildTypeDescFromType(obj.Type(), pkg)
	if desc.TypeName != "ApiReq" {
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "ApiReq")
	}
	if len(desc.Fields) < 2 {
		t.Fatalf("ApiReq should have at least 2 fields, got %d", len(desc.Fields))
	}
	// name field
	nameField := findField(desc.Fields, "name")
	if nameField == nil {
		t.Fatal("name field missing")
	}
	if nameField.TypeNameAlias != "string" {
		t.Errorf("name TypeNameAlias = %q, want %q", nameField.TypeNameAlias, "string")
	}
	// extras field ([]ApiReqExtra — nested struct slice)
	extrasField := findField(desc.Fields, "extras")
	if extrasField == nil {
		t.Fatal("extras field missing")
	}
	if !extrasField.IsSliceOrArray {
		t.Error("extras IsSliceOrArray should be true")
	}
	if len(extrasField.Fields) == 0 {
		t.Error("extras should have nested fields from ApiReqExtra")
	}
	// ApiReqExtra has a "special" field
	specialField := findField(extrasField.Fields, "special")
	if specialField == nil {
		t.Fatal("special field missing from nested ApiReqExtra")
	}
	if specialField.TypeNameAlias != "bool" {
		t.Errorf("special TypeNameAlias = %q, want %q", specialField.TypeNameAlias, "bool")
	}
}

func TestBuildTypeDescFromType_CustomAliasTypes(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	// PostRes2 has the full set of alias type fields including pointer variants
	obj := pkg.Scope().Lookup("PostRes2")
	if obj == nil {
		t.Fatal("PostRes2 not found")
	}
	desc := BuildTypeDescFromType(obj.Type(), pkg)

	// amt field: money.Amt → "string"
	amtField := findField(desc.Fields, "amt")
	if amtField == nil {
		t.Fatal("amt field missing")
	}
	if amtField.TypeNameAlias != "string" {
		t.Errorf("amt TypeNameAlias = %q, want %q (money.Amt → string)", amtField.TypeNameAlias, "string")
	}

	// amtPtr field: *money.Amt → "*string"
	amtPtrField := findField(desc.Fields, "amtPtr")
	if amtPtrField == nil {
		t.Fatal("amtPtr field missing")
	}
	if amtPtrField.TypeNameAlias != "*string" {
		t.Errorf("amtPtr TypeNameAlias = %q, want %q (*money.Amt → *string)", amtPtrField.TypeNameAlias, "*string")
	}

	// set field: hash.Set[string] → "[]string"
	setField := findField(desc.Fields, "set")
	if setField == nil {
		t.Fatal("set field missing")
	}
	if setField.TypeNameAlias != "[]string" {
		t.Errorf("set TypeNameAlias = %q, want %q (hash.Set[string] → []string)", setField.TypeNameAlias, "[]string")
	}

	// setPtr field: *hash.Set[string] → "[]string" (pointer stripped by alias)
	setPtrField := findField(desc.Fields, "setPtr")
	if setPtrField == nil {
		t.Fatal("setPtr field missing")
	}
	if setPtrField.TypeNameAlias != "[]string" {
		t.Errorf("setPtr TypeNameAlias = %q, want %q (*hash.Set[string] → []string)", setPtrField.TypeNameAlias, "[]string")
	}
}

// === resolveTypeRef direct tests ===

func TestResolveTypeRef_SamePkgValueType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{Name: "PostReq"}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName != "PostReq" {
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "PostReq")
	}
	if desc.IsPtr {
		t.Error("IsPtr should be false for value type")
	}
}

func TestResolveTypeRef_SamePkgPointerType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{Name: "PostReq", IsPtr: true}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName != "PostReq" {
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "PostReq")
	}
	if !desc.IsPtr {
		t.Error("IsPtr should be true")
	}
}

func TestResolveTypeRef_SamePkgSliceType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{Name: "ApiReq", IsSlice: true}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName != "ApiReq" {
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "ApiReq")
	}
	if !desc.IsSlice {
		t.Error("IsSlice should be true")
	}
}

func TestResolveTypeRef_CrossPkgType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	// Use a type from another package visible via imports
	ref := sourceparser.TypeRef{PkgName: "miso", Name: "Inbound"}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName == "" {
		t.Error("TypeName should be non-empty for miso.Inbound")
	}
}

func TestResolveTypeRef_BuiltInType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{Name: "string"}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName != "string" {
		t.Errorf("TypeName = %q, want %q", desc.TypeName, "string")
	}
}

func TestResolveTypeRef_MapType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{
		IsMap:    true,
		MapKey:   &sourceparser.TypeRef{Name: "string"},
		MapValue: &sourceparser.TypeRef{Name: "int32"},
	}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName == "" {
		t.Error("map type should have a TypeName")
	}
}

// === resolveTypeRef edge cases ===

func TestResolveTypeRef_NonExistentType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{Name: "NonExistentType"}
	desc := resolveTypeRef(ref, pkg, nil)
	// Should fallback to FullString
	if desc.TypeName != "NonExistentType" {
		t.Errorf("TypeName = %q, want fallback %q", desc.TypeName, "NonExistentType")
	}
}

func TestResolveTypeRef_NonExistentCrossPkgType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{PkgName: "nonexistent", Name: "Foo"}
	desc := resolveTypeRef(ref, pkg, nil)
	// Should fallback to FullString
	if desc.TypeName != "nonexistent.Foo" {
		t.Errorf("TypeName = %q, want fallback %q", desc.TypeName, "nonexistent.Foo")
	}
}

// === resolveGenericTypeRef edge cases ===

func TestResolveGenericTypeRef_PageResPostRes(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{
		PkgName:  "miso",
		Name:     "PageRes",
		TypeArgs: []sourceparser.TypeRef{{Name: "PostRes"}},
	}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName == "" {
		t.Error("PageRes[PostRes] should resolve to a TypeDesc")
	}
	if !strings.Contains(desc.TypeName, "PageRes") {
		t.Errorf("TypeName = %q, want to contain PageRes", desc.TypeName)
	}
	if !strings.Contains(desc.TypeName, "PostRes") {
		t.Errorf("TypeName = %q, want to contain PostRes", desc.TypeName)
	}
}

func TestResolveGenericTypeRef_WrongTypeParamsCount(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	// PageRes has 1 type param, give it 2
	ref := sourceparser.TypeRef{
		PkgName:  "miso",
		Name:     "PageRes",
		TypeArgs: []sourceparser.TypeRef{{Name: "string"}, {Name: "int"}},
	}
	desc := resolveTypeRef(ref, pkg, nil)
	// Should fall back to string representation (type param count mismatch)
	if desc.TypeName == "" {
		t.Error("should have a fallback type name")
	}
}

func TestResolveGenericTypeRef_NonExistentGenericType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{
		PkgName:  "nonexistent",
		Name:     "GenericType",
		TypeArgs: []sourceparser.TypeRef{{Name: "string"}},
	}
	desc := resolveTypeRef(ref, pkg, nil)
	// Should fall back
	if desc.TypeName == "" {
		t.Error("should have a fallback type name for non-existent generic type")
	}
}

func TestResolveGenericTypeRef_NonExistentBaseType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	ref := sourceparser.TypeRef{
		Name:     "NonExistentBase",
		TypeArgs: []sourceparser.TypeRef{{Name: "string"}},
	}
	desc := resolveTypeRef(ref, pkg, nil)
	if desc.TypeName == "" {
		t.Error("should have a fallback type name")
	}
}

// === lookupTypeInPkg tests ===

func TestLookupTypeInPkg_SamePackageType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	obj := lookupTypeInPkg(pkg, "PostRes")
	if obj == nil {
		t.Fatal("PostRes should be found in same package scope")
	}
	if obj.Name() != "PostRes" {
		t.Errorf("Name = %q, want %q", obj.Name(), "PostRes")
	}
}

func TestLookupTypeInPkg_CrossPackageType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	// Look up miso.Inbound via import
	obj := lookupTypeInPkg(pkg, "miso.Inbound")
	if obj == nil {
		t.Fatal("miso.Inbound should be found via imports")
	}
}

func TestLookupTypeInPkg_NonExistentType(t *testing.T) {
	pkg := loadDemoAPIPkg(t)
	obj := lookupTypeInPkg(pkg, "NonExistentType")
	if obj != nil {
		t.Error("NonExistentType should not be found")
	}
}

// === refTypeName and extractRequestParams tests ===

func TestRefTypeName(t *testing.T) {
	tests := []struct {
		ref  sourceparser.TypeRef
		want string
	}{
		{sourceparser.TypeRef{Name: "FileInfoReq"}, "FileInfoReq"},
		{sourceparser.TypeRef{PkgName: "api", Name: "FileInfoReq"}, "api.FileInfoReq"},
		{sourceparser.TypeRef{PkgName: "miso", Name: "Inbound", IsPtr: true}, "miso.Inbound"},
	}
	for _, tt := range tests {
		got := refTypeName(tt.ref)
		if got != tt.want {
			t.Errorf("refTypeName(%+v) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

func TestExtractRequestParams_ExplicitType_StillWorks(t *testing.T) {
	// explicit QueryReqType/HeaderReqType must not regress
	pkg := loadDemoAPIPkg(t)
	ep := &sourceparser.ParsedEndpoint{
		QueryReqType:  "PostReq",
		HeaderReqType: "PostReq",
	}
	doc := miso.HttpRouteDoc{}
	extractRequestParams(&doc, ep, pkg)

	// PostReq has no form or header tags
	if len(doc.QueryParams) != 0 {
		t.Errorf("expected 0 query params from PostReq (no form tags), got %d: %+v", len(doc.QueryParams), doc.QueryParams)
	}
	if len(doc.Headers) != 0 {
		t.Errorf("expected 0 headers from PostReq (no header tags), got %d: %+v", len(doc.Headers), doc.Headers)
	}
}

func TestExtractRequestParams_NilRequestRef(t *testing.T) {
	// nil RequestRef with empty QueryReqType/HeaderReqType must not panic
	pkg := loadDemoAPIPkg(t)
	ep := &sourceparser.ParsedEndpoint{
		QueryReqType:  "",
		HeaderReqType: "",
		RequestRef:    nil,
	}
	doc := miso.HttpRouteDoc{}
	extractRequestParams(&doc, ep, pkg)

	if len(doc.QueryParams) != 0 {
		t.Errorf("expected 0 query params, got %d", len(doc.QueryParams))
	}
	if len(doc.Headers) != 0 {
		t.Errorf("expected 0 headers, got %d", len(doc.Headers))
	}
}

func TestExtractRequestParams_RequestRefNoFormTags(t *testing.T) {
	// RequestRef set but type has no form/header tags — no params extracted
	pkg := loadDemoAPIPkg(t)
	ep := &sourceparser.ParsedEndpoint{
		QueryReqType:  "",
		HeaderReqType: "",
		RequestRef:    &sourceparser.TypeRef{Name: "PostReq"},
	}
	doc := miso.HttpRouteDoc{}
	extractRequestParams(&doc, ep, pkg)

	if len(doc.QueryParams) != 0 {
		t.Errorf("expected 0 query params from PostReq (no form tags), got %d: %+v", len(doc.QueryParams), doc.QueryParams)
	}
	if len(doc.Headers) != 0 {
		t.Errorf("expected 0 headers from PostReq (no header tags), got %d: %+v", len(doc.Headers), doc.Headers)
	}
}

func TestExtractRequestParams_FallbackToRequestRef(t *testing.T) {
	// When QueryReqType/HeaderReqType are empty, fall back to RequestRef
	// to extract form/header tags from the struct.
	dir := t.TempDir()

	goModPath := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module testpkg\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	srcPath := filepath.Join(dir, "pkg.go")
	srcContent := "package testpkg\n" +
		"\n" +
		"type FileInfoReq struct {\n" +
		"\tFileId       string `form:\"fileId\" desc:\"actual file_id of the file record\"`\n" +
		"\tUploadFileId string `form:\"uploadFileId\" desc:\"temporary file_id\"`\n" +
		"\tAuth         string `header:\"Authorization\" desc:\"bearer token\"`\n" +
		"}\n"
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatal(err)
	}

	pkg, err := loadPackageFromDir(nil, "testpkg", dir)
	if err != nil {
		t.Fatalf("loadPackageFromDir failed: %v", err)
	}

	ep := &sourceparser.ParsedEndpoint{
		QueryReqType:  "",
		HeaderReqType: "",
		RequestRef:    &sourceparser.TypeRef{Name: "FileInfoReq"},
	}
	doc := miso.HttpRouteDoc{}
	extractRequestParams(&doc, ep, pkg)

	// Query params from form tags
	if len(doc.QueryParams) != 2 {
		t.Errorf("expected 2 query params from form tags, got %d: %+v", len(doc.QueryParams), doc.QueryParams)
	}
	wantQueries := map[string]string{
		"fileId":       "actual file_id of the file record",
		"uploadFileId": "temporary file_id",
	}
	for _, q := range doc.QueryParams {
		expDesc, ok := wantQueries[q.Name]
		if !ok {
			t.Errorf("unexpected query param: %q", q.Name)
			continue
		}
		if q.Desc != expDesc {
			t.Errorf("query param %q desc = %q, want %q", q.Name, q.Desc, expDesc)
		}
	}

	// Header params from header tags
	if len(doc.Headers) != 1 {
		t.Errorf("expected 1 header from header tags, got %d: %+v", len(doc.Headers), doc.Headers)
	}
	if len(doc.Headers) > 0 {
		if doc.Headers[0].Name != "Authorization" {
			t.Errorf("header name = %q, want %q", doc.Headers[0].Name, "Authorization")
		}
		if doc.Headers[0].Desc != "bearer token" {
			t.Errorf("header desc = %q, want %q", doc.Headers[0].Desc, "bearer token")
		}
	}
}

// TestLoadPackageFromDir_SamePackageConsts verifies that packages.Load with
// NeedTypes|NeedImports preserves string consts in the loaded package's scope,
// enabling same-package const resolution via ParseFileDst.
func TestLoadPackageFromDir_SamePackageConsts(t *testing.T) {
	dir := t.TempDir()

	// go.mod
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`module testmod

go 1.21
`), 0644); err != nil {
		t.Fatal(err)
	}

	// pkg/consts.go — defines consts in the same package
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg", "handler.go"), []byte(`package pkg

const LocalURL = "/api/local"
const LocalDesc = "local description"

func init() {
	_ = LocalURL
	_ = LocalDesc
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Load the package
	pkg, err := loadPackageFromDir(nil, "testmod/pkg", filepath.Join(dir, "pkg"))
	if err != nil {
		t.Fatalf("loadPackageFromDir failed: %v", err)
	}

	// Verify LocalURL is accessible in package scope
	obj := pkg.Scope().Lookup("LocalURL")
	if obj == nil {
		t.Fatal("pkg.Scope().Lookup(\"LocalURL\") returned nil")
	}
	c, ok := obj.(*types.Const)
	if !ok {
		t.Fatalf("LocalURL is not a *types.Const, got %T", obj)
	}
	if c.Val() == nil || c.Val().Kind() != constant.String {
		t.Fatal("LocalURL.Val() is nil or not a string")
	}
	val := constant.StringVal(c.Val())
	if val != "/api/local" {
		t.Fatalf("LocalURL = %q, want %q", val, "/api/local")
	}
	t.Logf("LocalURL = %q ✓", val)
}
