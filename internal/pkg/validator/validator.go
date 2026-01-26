package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator instance
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Use JSON tag names in error messages
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register custom validations
	registerCustomValidations()
}

func registerCustomValidations() {
	// Role validation
	validate.RegisterValidation("role", func(fl validator.FieldLevel) bool {
		role := fl.Field().String()
		validRoles := []string{"model", "employer", "agency", "admin"}
		for _, r := range validRoles {
			if role == r {
				return true
			}
		}
		return false
	})

	// Gender validation
	validate.RegisterValidation("gender", func(fl validator.FieldLevel) bool {
		gender := fl.Field().String()
		validGenders := []string{"male", "female", "other", ""}
		for _, g := range validGenders {
			if gender == g {
				return true
			}
		}
		return false
	})

	// Pay type validation
	validate.RegisterValidation("pay_type", func(fl validator.FieldLevel) bool {
		payType := fl.Field().String()
		validTypes := []string{"fixed", "hourly", "negotiable", "free", ""}
		for _, t := range validTypes {
			if payType == t {
				return true
			}
		}
		return false
	})
}

// Validate validates a struct and returns a map of field errors
func Validate(s interface{}) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		switch err.Tag() {
		case "required":
			errors[field] = "This field is required"
		case "email":
			errors[field] = "Invalid email format"
		case "min":
			errors[field] = "Value is too short (min: " + err.Param() + ")"
		case "max":
			errors[field] = "Value is too long (max: " + err.Param() + ")"
		case "gte":
			errors[field] = "Value must be at least " + err.Param()
		case "lte":
			errors[field] = "Value must be at most " + err.Param()
		case "url":
			errors[field] = "Invalid URL format"
		case "role":
			errors[field] = "Invalid role. Must be: model, employer, agency, or admin"
		case "gender":
			errors[field] = "Invalid gender. Must be: male, female, or other"
		case "pay_type":
			errors[field] = "Invalid pay type. Must be: fixed, hourly, negotiable, or free"
		default:
			errors[field] = "Invalid value"
		}
	}

	return errors
}

// ValidateVar validates a single variable
func ValidateVar(field interface{}, tag string) error {
	return validate.Var(field, tag)
}
