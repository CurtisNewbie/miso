package miso

import (
	"fmt"

	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/strutil"
)

func skipGoParsingType(f interface {
	TypeInfo() (pkg string, typeName string)
}) bool {
	return FuzzMatchTypes(f, ApiDocGoSkipParsingTypes)
}

// generate one or more golang type definitions.
func GenGoDef(rv TypeDesc, seenTypeDef hash.Set[string]) (string, string) {
	if rv.TypeName == "any" || rv.TypeName == "interface{}" {
		return "", ""
	}

	if rv.TypeName == "Resp" || rv.TypeName == "GnResp" {
		for _, f := range rv.Fields {
			if f.GoFieldName == "Data" {
				if f.OriginTypeName == "any" || f.OriginTypeName == "interface{}" {
					return "", ""
				}
				deferred := make([]func(), 0, 10)
				sb, writef := strutil.NewIndWritef("\t")

				ptn := f.pureGoTypeName()

				if !skipGoParsingType(f) {
					inclTypeDef := inclGoTypeDef(f, seenTypeDef)
					if inclTypeDef {
						writef(0, "type %s struct {", ptn)
					}
					genGoDefRecur(1, writef, &deferred, f.Fields, inclTypeDef, seenTypeDef)
					if inclTypeDef {
						writef(0, "}")
					}
				}
				for i := 0; i < len(deferred); i++ {
					deferred[i]()
				}
				return sb.String(), ptn
			}
		}
		return "", ""
	} else {
		deferred := make([]func(), 0, 10)
		sb, writef := strutil.NewIndWritef("\t")
		ptn := rv.pureGoTypeName()

		if !skipGoParsingType(rv) {
			inclTypeDef := inclGoTypeDef(rv, seenTypeDef)
			if inclTypeDef {
				writef(0, "type %s struct {", ptn)
			}

			genGoDefRecur(1, writef, &deferred, rv.Fields, inclTypeDef, seenTypeDef)
			if inclTypeDef {
				writef(0, "}")
			}
		}

		for i := 0; i < len(deferred); i++ {
			deferred[i]()
		}
		return sb.String(), ptn
	}
}

func inclGoTypeDef(f interface {
	TypeInfo() (pkg string, typeName string)
	isBuiltInType() bool
	pureGoTypeName() string
}, seenTypeDef hash.Set[string]) bool {

	if f.isBuiltInType() { // e.g., map
		return false
	}

	pgn := f.pureGoTypeName()
	p, n := f.TypeInfo()
	Debugf("inclGoTypeDef: %v, %v, %v\n", pgn, p, n)

	// TODO: temp fix
	switch pgn {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "string", "bool", "byte":
		return false
	}

	if FuzzMatchTypes(f, ApiDocNotInclTypes) {
		return false
	}

	if !seenTypeDef.Add(pgn) {
		return false
	}
	return true
}

func genGoDefRecur(indentc int, writef strutil.IndWritef, deferred *[]func(), fields []FieldDesc, writeField bool,
	seenTypeDef hash.Set[string]) {

	for _, f := range fields {
		var jsonTag string
		if f.JsonTag != "" {
			jsonTag = fmt.Sprintf(" `json:\"%v\"`", f.JsonTag)
		}
		ffields := f.Fields

		if len(ffields) > 0 {

			if writeField {
				fieldTypeName := f.goFieldTypeName()
				writef(indentc, "%s %s%s", f.GoFieldName, fieldTypeName, jsonTag)
			}

			if !skipGoParsingType(f) {
				inclType := inclGoTypeDef(f, seenTypeDef)
				*deferred = append(*deferred, func() {
					if inclType {
						writef(0, "")
						writef(0, "type %s struct {", f.pureGoTypeName())
					}
					genGoDefRecur(1, writef, deferred, f.Fields, inclType, seenTypeDef)
					if inclType {
						writef(0, "}")
					}
				})
			}

		} else {
			if !writeField {
				continue
			}
			fieldTypeName := f.goFieldTypeName()
			var comment string = f.comment(true)
			if comment != "" {
				fieldDec := fmt.Sprintf("%s %s%s", f.GoFieldName, fieldTypeName, jsonTag)
				writef(indentc, "%-30s%s", fieldDec, comment)
			} else {
				writef(indentc, "%s %s%s", f.GoFieldName, fieldTypeName, jsonTag)
			}
		}
	}
}
