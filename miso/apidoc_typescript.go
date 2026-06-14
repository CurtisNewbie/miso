package miso

import (
	"fmt"
	"strings"

	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/strutil"
)

// generate one or more typescript interface definitions based on a set of jsonDesc.
func GenTsDef(payload TypeDesc) string {
	var typeName string = payload.TypeName
	if len(payload.Fields) < 1 && typeName == "" {
		return ""
	}
	sb, writef := strutil.NewIndWritef("  ")
	seenType := hash.NewSet[string]()
	tsTypeName := guessTsItfName(typeName)
	seenType.Add(tsTypeName)
	writef(0, "export interface %s {", tsTypeName)
	deferred := make([]func(), 0, 10)
	genTsDefRecur(1, writef, true, &deferred, payload.Fields, seenType)
	writef(0, "}")

	for i := 0; i < len(deferred); i++ {
		writef(0, "")
		deferred[i]()
	}
	return sb.String()
}

func genTsDefRecur(indentc int, writef strutil.IndWritef, writeField bool, deferred *[]func(), descs []FieldDesc, seenType hash.Set[string]) {
	for i := range descs {
		d := descs[i]

		if len(d.Fields) > 0 {
			tsTypeName := guessTsItfName(d.TypeNameAlias)
			if writeField {
				n := tsTypeName
				if strings.HasPrefix(d.TypeNameAlias, "[]") {
					n += "[]"
				}
				writef(indentc, "%s?: %s;", d.JsonName, n)
			}

			// TODO: this is ugly
			inclType := seenType.Add(tsTypeName)
			stopDesc := false
			if inclType {
				if FuzzMatchTypes(d, ApiDocTsSkipParsingTypes) {
					inclType = false
					stopDesc = true
				}
			}
			if !stopDesc {
				*deferred = append(*deferred, func() {
					if inclType {
						writef(0, "export interface %s {", tsTypeName)
					}
					genTsDefRecur(1, writef, inclType, deferred, d.Fields, seenType)
					if inclType {
						writef(0, "}")
					}
				})
			}
		} else if writeField {
			var tname string = d.guessTsPrimiTypeName()
			var comment string = d.comment(true)
			if comment != "" {
				fieldDec := fmt.Sprintf("%s?: %s", d.JsonName, tname)
				writef(indentc, "%-30s%s", fieldDec+";", comment)
			} else {
				writef(indentc, "%s?: %s;", d.JsonName, tname)
			}
		}
	}
}

// try to convert golang type name to typescript primitive type name.
func guessTsPrimiTypeName(typeName string) string {
	var tname string
	switch typeName {
	case "string", "*string":
		tname = "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"*int", "*int8", "*int16", "*int32", "*int64", "*uint", "*uint8", "*uint16", "*uint32", "*uint64":
		tname = "number"
	case "float32", "float64", "*float32", "*float64":
		tname = "number"
	case "bool", "*bool":
		tname = "boolean"
	default:
		if v, ok := strings.CutPrefix(typeName, "[]"); ok {
			tname = guessTsItfName(v) + "[]"
		} else {
			tname = guessTsItfName(typeName)
		}
	}
	return tname
}

// try to convert golang type (incl struct name) name to typescript interface name.
func guessTsItfName(n string) string {
	if len(n) == 0 {
		return n
	}

	// *MyType -> MyType (Go pointer, no TS equivalent)
	if v, ok := strings.CutPrefix(n, "*"); ok {
		n = v
	}

	// cp := n
	v, ok := strings.CutPrefix(n, "[]")
	if ok {
		if len(n) == 2 {
			return n
		}
		n = v
	}

	if n[len(n)-1] == ']' {
		j := strings.IndexByte(n, '[')
		if j > -1 {
			n = n[:j]
		}
	}

	i := strings.LastIndexByte(n, '.')
	if i > -1 {
		n = n[i+1:]
	}
	// Debugf("guessing typescript interface name: %v -> %v", cp, n)
	return n
}

func guessGoGenericEleName(n string) string {
	if len(n) < 3 {
		return ""
	}
	if n[len(n)-1] != ']' {
		return ""
	}
	i := strings.IndexByte(n, '[')
	if i < 0 {
		return ""
	}
	v := n[i+1 : len(n)-1]
	return guessGoTypName(v)
}

func guessGoTypName(n string) string {
	tsTypeName := guessTsItfName(n)
	return tsTypeName
}

func guessTsTypeName(d FieldDesc) string {
	if len(d.Fields) > 0 {
		tsTypeName := guessTsItfName(d.TypeNameAlias)
		if strings.HasPrefix(d.TypeNameAlias, "[]") {
			return tsTypeName + "[]"
		}
		return tsTypeName
	} else {
		return d.guessTsPrimiTypeName()
	}
}
