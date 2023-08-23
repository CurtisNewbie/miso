package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	TAG_VALIDATION_V1 = "validation" // name of validation tag
	TAG_VALIDATION_V2 = "valid"      // name of validation tag (v2)

	MAX_LEN = "maxLen" // max length of a string, array, slice (e.g., 'maxLen:10')

	NOT_EMPTY = "notEmpty" // not empty, supports string, array, slice, map
	NOT_NIL   = "notNil"   // not nil, only validates slice, map, pointer, func

	POSITIVE         = "positive"       // greater than 0, only supports int... or string type
	POSITIVE_OR_ZERO = "positiveOrZero" // greater than or equal to 0, only supports int... or string type
	NEGATIVE         = "negative"       // less than 0, only supports int... or string type
	NEGATIVE_OR_ZERO = "negativeOrZero" // less than or equal to 0, only supports int... or string type
	NOT_ZERO         = "notZero"        // not zero, only supports int... or string type
	VALIDATED        = "validated"      // mark a nested struct or pointer validated, nil pointer is ignored, one may combine "notNil,validated"
)

var (
	rules = NewSet[string]()
)

func init() {
	rules.AddThen(MAX_LEN).
		AddThen(NOT_EMPTY).
		AddThen(NOT_NIL).
		AddThen(POSITIVE).
		AddThen(POSITIVE_OR_ZERO).
		AddThen(NEGATIVE).
		AddThen(NEGATIVE_OR_ZERO).
		AddThen(NOT_ZERO).
		Add(VALIDATED)
}

// Validation Error
type ValidationError struct {
	Field         string
	Rule          string
	ValidationMsg string
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
	introspector := Introspect(target)
	targetVal := reflect.ValueOf(target)
	var verr error

	forEach := func(i int, field reflect.StructField) (breakIteration bool) {
		vtag := field.Tag.Get(TAG_VALIDATION_V2) // new tag

		if vtag == "" {
			vtag = field.Tag.Get(TAG_VALIDATION_V1) // old tag
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
	if !IsFieldExposed(fname) {
		return nil
	}

	// logrus.Infof("Validating '%s' with value '%v' against rule '%s'", fname, value, rule)

	switch rule {
	case MAX_LEN:
		maxLen, e := strconv.Atoi(ruleParam)
		if e == nil && maxLen > -1 {
			switch value.Kind() {
			case reflect.Slice, reflect.Array:
				currLen := value.Len()
				if currLen > maxLen {
					return &ValidationError{Field: fname, Rule: rule, ValidationMsg: fmt.Sprintf("exceeded maximum length %d, current length: %d", maxLen, currLen)}
				}
			case reflect.String:
				currLen := utf8.RuneCountInString(value.String())
				if currLen > maxLen {
					return &ValidationError{Field: fname, Rule: rule, ValidationMsg: fmt.Sprintf("exceeded maximum length %d, current length: %d", maxLen, currLen)}
				}
			}
		}
	case NOT_EMPTY:
		switch value.Kind() {
		case reflect.String:
			sval := value.String()
			if trimed := strings.TrimSpace(sval); trimed == "" {
				return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must not be empty"}
			}
		case reflect.Array, reflect.Slice, reflect.Map:
			if value.Len() < 1 {
				return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must not be empty"}
			}
		}
	case NOT_NIL:
		switch value.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func:
			if value.IsNil() {
				return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "cannot be nil"}
			}
		}
	case POSITIVE, POSITIVE_OR_ZERO, NEGATIVE, NEGATIVE_OR_ZERO, NOT_ZERO:
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
	case VALIDATED:
		switch value.Kind() {
		case reflect.Struct:
			// nested struct, validate recursively
			return ChainValidationError(fname, Validate(value.Interface()))
		case reflect.Pointer:
			// not nil pointer, dereference and validate recursively
			if !value.IsNil() {
				return ChainValidationError(fname, Validate(value.Elem().Interface()))
			}
		}
	}
	return nil
}

func ValidateIntRule(ival int64, rule string, fname string, param string) error {
	switch rule {
	case POSITIVE:
		if ival <= 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be greater than zero"}
		}
	case POSITIVE_OR_ZERO:
		if ival < 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be grater than or equal to zero"}
		}
	case NEGATIVE:
		if ival >= 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be less than zero"}
		}
	case NEGATIVE_OR_ZERO:
		if ival > 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must be less than or equal to zero"}
		}
	case NOT_ZERO:
		if ival == 0 {
			return &ValidationError{Field: fname, Rule: rule, ValidationMsg: "must not be zero"}
		}
	}
	return nil
}
