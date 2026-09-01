package validator

import (
	"fmt"
	"reflect"
	"strings"

	govalidator "github.com/go-playground/validator/v10"
)

var validate *govalidator.Validate

func init() {
	validate = govalidator.New()

	// Register tag nameFunc to use json tag names instead of struct field names
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// Validate validates a struct and returns field-level error mapping or nil if valid.
func Validate(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errs := make(map[string]string)
	if valErrs, ok := err.(govalidator.ValidationErrors); ok {
		for _, e := range valErrs {
			field := e.Field()
			errs[field] = formatErrorMessage(e)
		}
	} else {
		errs["_global"] = err.Error()
	}

	return errs
}

func formatErrorMessage(e govalidator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", e.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", e.Field(), e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", e.Field(), e.Param())
	case "nefield":
		return fmt.Sprintf("%s cannot be identical to %s", e.Field(), e.Param())
	default:
		return fmt.Sprintf("%s failed on '%s' validation rule", e.Field(), e.Tag())
	}
}
