package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Validator is implemented by request DTOs that support validation.
// Validate returns a slice of error messages; nil or empty means valid.
type Validator interface {
	Validate() []string
}

// DecodeAndValidate decodes the request body into dest (with DisallowUnknownFields)
// and, if dest implements Validator, runs Validate(). On decode or validation failure
// it writes a 400 JSON error and returns false; otherwise returns true.
// Callers should return immediately when DecodeAndValidate returns false.
func DecodeAndValidate(w http.ResponseWriter, r *http.Request, dest any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		WriteJSONError(w, http.StatusBadRequest, ErrCodeBadRequest, err.Error())
		return false
	}
	if v, ok := dest.(Validator); ok {
		if errs := v.Validate(); len(errs) > 0 {
			WriteJSONError(w, http.StatusBadRequest, ErrCodeBadRequest, strings.Join(errs, "; "))
			return false
		}
	}
	return true
}
