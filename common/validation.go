package common

import (
	"reflect"
	"strconv"
	"strings"
)

const (
	TAG_VALIDATION = "validation" // name of validation tag

	NOT_EMPTY = "notEmpty" // not empty, supports string, array, slice, map, chan
	NOT_NIL   = "notNil"   // not nil, only validates pointer

	/*
		-----------------------------

		For Numbers

		-----------------------------
	*/

	POSITIVE         = "positive"       // greater than 0, only supports int... or string type
	POSITIVE_OR_ZERO = "positiveOrZero" // greater than or equal to 0, only supports int... or string type
	NEGATIVE         = "negative"       // less than 0, only supports int... or string type
	NEGATIVE_OR_ZERO = "negativeOrZero" // less than or equal to 0, only supports int... or string type
	NOT_ZERO         = "notZero"        // not zero, only supports int... or string type
)

var (
	rules = NewSet[string]()
)

func init() {
	rules.Add(NOT_EMPTY)
	rules.Add(NOT_NIL)
	rules.Add(POSITIVE)
	rules.Add(POSITIVE_OR_ZERO)
	rules.Add(NEGATIVE)
	rules.Add(NEGATIVE_OR_ZERO)
	rules.Add(NOT_ZERO)
}

// Validation Error
type ValidationError struct {
	Field         string
	ValidationMsg string
}

func (ve *ValidationError) Error() string {
	return ve.Field + " " + ve.ValidationMsg
}

// Validate target object based on the validation rules specified by tags
func Validate(target any) error {
	introspector := Introspect(target)
	targetVal := reflect.ValueOf(target)
	var verr error

	forEach := func(i int, field reflect.StructField) (breakIteration bool) {
		vtag := field.Tag.Get(TAG_VALIDATION)
		if vtag != "" {
			validations := strings.Split(vtag, ",")
			fval := targetVal.Field(i)

			for _, v := range validations {
				if rules.Has(v) {
					if e := ValidateRule(field, fval, v); e != nil {
						verr = e
						return true
					}
				}
			}
		}
		return false
	}

	introspector.IterFields(forEach)
	return verr
}

// Validate field against the rule
func ValidateRule(field reflect.StructField, value reflect.Value, rule string) error {
	fname := field.Name
	if !IsFieldExposed(fname) {
		return nil
	}

	// logrus.Infof("Validating '%s' with value '%v' against rule '%s'", fname, value, rule)

	switch value.Kind() {
	case reflect.Pointer:
		if rule == NOT_NIL && value.IsNil() {
			return &ValidationError{Field: fname, ValidationMsg: "cannot be nil"}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ival := value.Int()
		return ValidateIntRule(ival, rule, fname)
	case reflect.String:
		sval := value.String()
		switch rule {
		case NOT_EMPTY:
			if trimed := strings.TrimSpace(sval); trimed == "" {
				return &ValidationError{Field: fname, ValidationMsg: "must not be empty"}
			}
		case POSITIVE, POSITIVE_OR_ZERO, NEGATIVE, NEGATIVE_OR_ZERO:
			ival, e := strconv.Atoi(sval)
			if e != nil {
				return &ValidationError{Field: fname, ValidationMsg: "is not an integer"}
			}
			return ValidateIntRule(int64(ival), rule, fname)
		}
	case reflect.Array:
		switch rule {
		case NOT_EMPTY:
			if value.Len() < 1 {
				return &ValidationError{Field: fname, ValidationMsg: "must not be empty"}
			}
		}
	case reflect.Slice:
		switch rule {
		case NOT_EMPTY:
			if value.Len() < 1 {
				return &ValidationError{Field: fname, ValidationMsg: "must not be empty"}
			}
		}
	case reflect.Map:
		switch rule {
		case NOT_EMPTY:
			if value.Len() < 1 {
				return &ValidationError{Field: fname, ValidationMsg: "must not be empty"}
			}
		}
	case reflect.Chan:
		switch rule {
		case NOT_EMPTY:
			if value.Len() < 1 {
				return &ValidationError{Field: fname, ValidationMsg: "must not be empty"}
			}
		}
	}
	return nil
}

func ValidateIntRule(ival int64, rule string, fname string) error {
	switch rule {
	case POSITIVE:
		if ival <= 0 {
			return &ValidationError{Field: fname, ValidationMsg: "must be greater than zero"}
		}
	case POSITIVE_OR_ZERO:
		if ival < 0 {
			return &ValidationError{Field: fname, ValidationMsg: "must be grater than or equal to zero"}
		}
	case NEGATIVE:
		if ival >= 0 {
			return &ValidationError{Field: fname, ValidationMsg: "must be less than zero"}
		}
	case NEGATIVE_OR_ZERO:
		if ival > 0 {
			return &ValidationError{Field: fname, ValidationMsg: "must be less than or equal to zero"}
		}
	case NOT_ZERO:
		if ival == 0 {
			return &ValidationError{Field: fname, ValidationMsg: "must not be zero"}
		}
	}
	return nil
}
