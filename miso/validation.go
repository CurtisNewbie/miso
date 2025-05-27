package miso

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/curtisnewbie/miso/util"
)

const (
	TagValidationV1 = "validation" // name of validation tag
	TagValidationV2 = "valid"      // name of validation tag (v2)

	ValidMaxLen = "maxLen" // max length of a string, array, slice, e.g., `valid:"maxLen:10"`

	ValidNotEmpty = "notEmpty" // not empty, e.g., `valid:"notEmpty"` or `valid:"notEmpty:MyField is required"`. Supports string, array, slice, map.
	ValidNotNil   = "notNil"   // not nil, e.g., `valid:"notNil"` or `valid:"notNil:MyField is required"`. Only validates slice, map, pointer, func.

	// must be one of the values listed, e.g., 'valid:"member:PUBLIC|PROTECTED"', means that the tag value must be either PUBLIC or PROTECTED.
	// only string type is supported.
	ValidMember = "member"

	ValidPositive       = "positive"       // greater than 0, only supports int... or string type
	ValidPositiveOrZero = "positiveOrZero" // greater than or equal to 0, only supports int... or string type
	ValidNegative       = "negative"       // less than 0, only supports int... or string type
	ValidNegativeOrZero = "negativeOrZero" // less than or equal to 0, only supports int... or string type
	ValidNotZero        = "notZero"        // not zero, only supports int... or string type
	Validated           = "validated"      // mark a nested struct or pointer validated, nil pointer is ignored, one may combine "notNil,validated"
	ValidTrim           = "trim"           // trim string and *string value
)

var (
	rules = util.NewSet[string](
		ValidMaxLen,
		ValidNotEmpty,
		ValidNotNil,
		ValidMember,
		ValidPositive,
		ValidPositiveOrZero,
		ValidNegative,
		ValidNegativeOrZero,
		ValidNotZero,
		Validated,
		ValidTrim,
	)
	ValidateWalkTagCallbackDeprecated = util.WalkTagCallback{
		Tag:      TagValidationV1,
		OnWalked: validateOnWalked,
	}
	ValidateWalkTagCallback = util.WalkTagCallback{
		Tag:      TagValidationV2,
		OnWalked: validateOnWalked,
	}
)

// Validation Error
type ValidationError struct {
	Field               string
	Rule                string
	ValidationMsg       string
	CustomValidationMsg string
}

func ChainValidationError(parentField string, e error) error {
	if e == nil {
		return nil
	}

	if ve, ok := e.(*ValidationError); ok {
		return &ValidationError{Field: parentField + "." + ve.Field, Rule: ve.Rule, ValidationMsg: ve.ValidationMsg}
	}
	return e
}

func (ve *ValidationError) Error() string {
	if ve.CustomValidationMsg != "" {
		return ve.CustomValidationMsg
	}
	return ve.Field + " " + ve.ValidationMsg
}

/*
Validate target object based on the validation rules specified by tags 'valid:"[RULE]"'.

Available Rules:

  - maxLen
  - notEmpty
  - notNil
  - positive
  - positiveOrZero
  - negative
  - negativeOrZero
  - notZero
  - validated
*/
func Validate(target any) error {
	if rv, ok := target.(reflect.Value); ok {
		target = rv.Interface() // unwrap first
	}

	introspector := util.Introspect(target)
	targetVal := reflect.ValueOf(target)
	var verr error

	forEach := func(i int, field reflect.StructField) (breakIteration bool) {
		vtag := field.Tag.Get(TagValidationV2) // new tag

		if vtag == "" {
			vtag = field.Tag.Get(TagValidationV1) // old tag
		}

		if vtag == "" { // no tag found
			return false
		}

		taggedRules := strings.Split(vtag, ",")
		fval := targetVal.Field(i)

		// for each rule
		for _, rul := range taggedRules {
			rul = strings.TrimSpace(rul)

			// the tagged rule may contain extra parameters, e.g., 'maxLen:10'
			splited := strings.Split(rul, ":")
			for i := range splited {
				splited[i] = strings.TrimSpace(splited[i])
			}

			rul = splited[0] // rule is the one before ':'
			param := ""      // param is those joined after the first ':'

			if len(splited) > 1 { // contains extra parameters
				param = strings.Join(splited[1:], ":")
			}

			if rules.Has(rul) { // is a valid rule
				if e := ValidateRule(field, fval, rul, param); e != nil {
					verr = e
					return true
				}
			}
		}
		return false
	}

	introspector.IterFields(forEach)
	return verr
}

func ValidateRule(field reflect.StructField, value reflect.Value, rule string, ruleParam string) error {
	fname := field.Name
	if !util.IsFieldExposed(fname) {
		return nil
	}

	// Infof("Validating '%s' with value '%v' against rule '%s'", fname, value, rule)

	switch rule {
	case ValidMaxLen:
		maxLen, e := strconv.Atoi(ruleParam)
		if e == nil && maxLen > -1 {
			switch value.Kind() {
			case reflect.Slice, reflect.Array:
				currLen := value.Len()
				if currLen > maxLen {
					return &ValidationError{Field: fname, Rule: rule,
						ValidationMsg: fmt.Sprintf("exceeded maximum length %d, current length: %d", maxLen, currLen)}
				}
			case reflect.String:
				currLen := utf8.RuneCountInString(value.String())
				if currLen > maxLen {
					return &ValidationError{Field: fname, Rule: rule,
						ValidationMsg: fmt.Sprintf("exceeded maximum length %d, current length: %d", maxLen, currLen)}
				}
			}
		}
	case ValidNotEmpty:
		customMsg := strings.TrimSpace(ruleParam)
		switch value.Kind() {
		case reflect.String:
			sval := value.String()
			if util.IsBlankStr(sval) {
				return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must not be empty",
					CustomValidationMsg: customMsg}
			}
		case reflect.Array, reflect.Slice, reflect.Map:
			if value.Len() < 1 {
				return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must not be empty",
					CustomValidationMsg: customMsg}
			}
		}
	case ValidNotNil:
		customMsg := strings.TrimSpace(ruleParam)
		switch value.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func:
			if value.IsNil() {
				return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "cannot be nil", CustomValidationMsg: customMsg}
			}
		}
	case ValidMember:
		members := strings.Split(ruleParam, "|")
		if len(members) < 1 {
			return nil
		}
		switch value.Kind() {
		case reflect.String:
			sval := value.String()
			trimed := strings.TrimSpace(sval)
			for _, v := range members {
				if v == trimed {
					return nil
				}
			}
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: fmt.Sprintf("must be one of %v", members)}
		}
	case ValidPositive, ValidPositiveOrZero, ValidNegative, ValidNegativeOrZero, ValidNotZero:
		switch value.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return ValidateIntRule(value.Int(), rule, fname, ruleParam)
		case reflect.String:
			ival, e := strconv.Atoi(value.String())
			if e != nil {
				return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "is not an integer"}
			}
			return ValidateIntRule(int64(ival), rule, fname, ruleParam)
		}
	case Validated:
		switch value.Kind() {
		case reflect.Struct:
			// nested struct, validate recursively
			return ChainValidationError(fname, Validate(value.Interface()))
		case reflect.Pointer:
			// not nil pointer, dereference and validate recursively
			if !value.IsNil() {
				return ChainValidationError(fname, Validate(value.Elem().Interface()))
			}
		case reflect.Slice:
			return validateSlice(fname, value)
		case reflect.Array:
			value = value.Slice(0, value.Len())
			return validateSlice(fname, value)
		}
	case ValidTrim:
		switch value.Kind() {
		case reflect.String:
			value.SetString(strings.TrimSpace(value.String()))
			return nil
		case reflect.Pointer:
			if value.IsNil() {
				return nil
			}
			v := value.Elem().Interface()
			if s, ok := v.(string); ok {
				value.Elem().SetString(strings.TrimSpace(s))
			}
		}
	}
	return nil
}

func validateSlice(fname string, value reflect.Value) error {
	for i := 0; i < value.Len(); i++ {
		el := value.Index(i)
		if err := Validate(el.Interface()); err != nil {
			return ChainValidationError(fname, err)
		}
	}
	return nil
}

func ValidateIntRule(ival int64, rule string, fname string, param string) error {
	switch rule {
	case ValidPositive:
		if ival <= 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be greater than zero"}
		}
	case ValidPositiveOrZero:
		if ival < 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be grater than or equal to zero"}
		}
	case ValidNegative:
		if ival >= 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be less than zero"}
		}
	case ValidNegativeOrZero:
		if ival > 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be less than or equal to zero"}
		}
	case ValidNotZero:
		if ival == 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must not be zero"}
		}
	}
	return nil
}

func validateOnWalked(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
	taggedRules := strings.Split(tagVal, ",")

	// for each rule
	for _, rul := range taggedRules {
		rul = strings.TrimSpace(rul)

		// the tagged rule may contain extra parameters, e.g., 'maxLen:10'
		splited := strings.Split(rul, ":")
		for i := range splited {
			splited[i] = strings.TrimSpace(splited[i])
		}

		rul = splited[0] // rule is the one before ':'
		param := ""      // param is those joined after the first ':'

		if len(splited) > 1 { // contains extra parameters
			param = strings.Join(splited[1:], ":")
		}

		if rules.Has(rul) { // is a valid rule
			if err := ValidateRule(fieldType, fieldVal, rul, param); err != nil {
				return err
			}
		}
	}
	return nil
}
