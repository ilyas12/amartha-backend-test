package http

import "strings"

// ---- helpers ----

func containsFieldMsg(list []FieldError, field, substr string) bool {
	for _, e := range list {
		if e.Field == field && strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}
