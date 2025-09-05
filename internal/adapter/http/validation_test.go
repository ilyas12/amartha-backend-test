package http

import (
	"errors"
	"strings"
	"testing"
)

func TestHex32Validation(t *testing.T) {
	type P struct {
		BorrowerID string `validate:"hex32"`
	}
	cv := NewValidator()

	// valid: 32-char lowercase hex
	ok := P{BorrowerID: strings.Repeat("a", 32)}
	if err := cv.Validate(ok); err != nil {
		t.Fatalf("expected valid hex32, got err: %v", err)
	}

	// invalid samples
	for _, s := range []string{
		"",                                  // empty
		strings.Repeat("A", 32),             // uppercase
		"deadbeef",                          // too short
		strings.Repeat("g", 32),             // non-hex char
		"3f9a6a1b3d544fbe8b3a6b3e8d6b2c8",   // 31 chars
		"3f9a6a1b3d544fbe8b3a6b3e8d6b2c88x", // 33 with extra
	} {
		bad := P{BorrowerID: s}
		err := cv.Validate(bad)
		if err == nil {
			t.Fatalf("expected error for %q", s)
		}
		fe := ToFieldErrors(err)
		found := false
		for _, e := range fe {
			if e.Field == "BorrowerID" && strings.Contains(e.Message, "32-char lowercase hex") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected hex32 message for %q, got: %+v", s, fe)
		}
	}
}

func TestIntLikeValidation(t *testing.T) {
	type P struct {
		Amount float64 `validate:"intlike"`
	}
	cv := NewValidator()

	for _, v := range []float64{0, 5_000_000, 100_000_000, 123.0} {
		if err := cv.Validate(P{Amount: v}); err != nil {
			t.Fatalf("expected intlike OK for %v, got %v", v, err)
		}
	}
	for _, v := range []float64{1.1, 5_000_000.01, -3.14} {
		err := cv.Validate(P{Amount: v})
		if err == nil {
			t.Fatalf("expected intlike error for %v", v)
		}
		fe := ToFieldErrors(err)
		if !containsFieldMsg(fe, "Amount", "integer value") {
			t.Fatalf("expected 'integer value' for %v, got %+v", v, fe)
		}
	}
}

func TestDec2Validation(t *testing.T) {
	type P struct {
		Rate float64 `validate:"dec2"`
	}
	cv := NewValidator()

	for _, v := range []float64{1.29, 2.00, 0.9, 1.2} {
		if err := cv.Validate(P{Rate: v}); err != nil {
			t.Fatalf("expected dec2 OK for %v, got %v", v, err)
		}
	}
	for _, v := range []float64{1.234, 2.9999} {
		err := cv.Validate(P{Rate: v})
		if err == nil {
			t.Fatalf("expected dec2 error for %v", v)
		}
		fe := ToFieldErrors(err)
		if !containsFieldMsg(fe, "Rate", "at most 2 decimal places") {
			t.Fatalf("expected 'at most 2 decimal places' for %v, got %+v", v, fe)
		}
	}
}

func TestRequiredAndBoundsMapping(t *testing.T) {
	type P struct {
		Name string  `validate:"required"`
		Min  int     `validate:"gte=10"`
		Max  int     `validate:"lte=5"`
		ROI  float64 `validate:"dec2,gte=0.90,lte=1.29"`
	}
	cv := NewValidator()

	// Intentionally violate all
	err := cv.Validate(P{
		Name: "",    // required
		Min:  9,     // gte=10
		Max:  6,     // lte=5
		ROI:  1.333, // dec2 + lte fail, but dec2 will trigger first
	})
	if err == nil {
		t.Fatalf("expected validation errors")
	}
	fe := ToFieldErrors(err)

	// required
	if !containsFieldMsg(fe, "Name", "is required") {
		t.Fatalf("missing 'is required' for Name: %+v", fe)
	}
	// gte
	if !containsFieldMsg(fe, "Min", "greater than or equal to 10") {
		t.Fatalf("missing gte message for Min: %+v", fe)
	}
	// lte
	if !containsFieldMsg(fe, "Max", "less than or equal to 5") {
		t.Fatalf("missing lte message for Max: %+v", fe)
	}
	// dec2 mapping should show for ROI
	if !containsFieldMsg(fe, "ROI", "at most 2 decimal places") {
		t.Fatalf("missing dec2 message for ROI: %+v", fe)
	}
}

func TestToFieldErrors_NonValidation(t *testing.T) {
	err := errors.New("boom")
	fe := ToFieldErrors(err)
	if len(fe) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(fe))
	}
	if fe[0].Field != "_" || fe[0].Message != "boom" {
		t.Fatalf("unexpected mapping: %+v", fe[0])
	}
}
