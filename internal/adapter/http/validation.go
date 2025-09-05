package http

import (
	"math"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// Reusable error payload
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
type ErrorResponse struct {
	Error   string       `json:"error"`
	Details []FieldError `json:"details,omitempty"`
}

var reHex32 = regexp.MustCompile(`^[a-f0-9]{32}$`)

type CustomValidator struct{ v *validator.Validate }

func NewValidator() *CustomValidator {
	v := validator.New()

	// borrower id = 32-char lowercase hex
	_ = v.RegisterValidation("hex32", func(fl validator.FieldLevel) bool {
		return reHex32.MatchString(fl.Field().String())
	})
	// principal must be "integer-like" even if float64
	_ = v.RegisterValidation("intlike", func(fl validator.FieldLevel) bool {
		f := fl.Field().Float()
		return math.Abs(f-math.Round(f)) < 1e-9
	})
	// max 2 decimal places
	_ = v.RegisterValidation("dec2", func(fl validator.FieldLevel) bool {
		f := fl.Field().Float()
		return math.Abs(f-(math.Round(f*100)/100)) < 1e-9
	})

	return &CustomValidator{v: v}
}

func (cv *CustomValidator) Validate(i any) error { return cv.v.Struct(i) }

// Map validator.ValidationErrors → []FieldError with readable messages.
func ToFieldErrors(err error) []FieldError {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return []FieldError{{Field: "_", Message: err.Error()}}
	}
	out := make([]FieldError, 0, len(ve))
	for _, e := range ve {
		field := e.Field() // you can map to json tag if you prefer
		switch e.Tag() {
		case "required":
			out = append(out, FieldError{Field: field, Message: "is required"})
		case "hex32":
			out = append(out, FieldError{Field: field, Message: "must be 32-char lowercase hex"})
		case "intlike":
			out = append(out, FieldError{Field: field, Message: "must be an integer value"})
		case "dec2":
			out = append(out, FieldError{Field: field, Message: "must have at most 2 decimal places"})
		case "gte":
			out = append(out, FieldError{Field: field, Message: "must be greater than or equal to " + e.Param()})
		case "lte":
			out = append(out, FieldError{Field: field, Message: "must be less than or equal to " + e.Param()})
		default:
			out = append(out, FieldError{Field: field, Message: e.Tag() + " validation failed"})
		}
	}
	return out
}
