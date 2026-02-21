package http

import (
	"encoding/json"
	"net/http"
)

// Error codes for API error responses. Use these with WriteJSONError.
const (
	ErrCodeBadRequest    = "bad_request"
	ErrCodeUnauthorized  = "unauthorized"
	ErrCodeInternalError = "internal_error"
)

// APIError is the error object in the standardized API response envelope.
// swagger:model APIError
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIResponse is the standardized envelope for all API responses.
// On success: Data is set, Error is nil. On error: Data is nil, Error is set.
// swagger:model APIResponse
type APIResponse struct {
	Data  any       `json:"data"`
	Error *APIError `json:"error"`
}

// WriteJSONSuccess sets Content-Type to application/json, writes statusCode, and
// encodes an APIResponse with the given data and error set to nil.
func WriteJSONSuccess(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(APIResponse{Data: data, Error: nil})
}

// WriteJSONError sets Content-Type to application/json, writes statusCode, and
// encodes an APIResponse with data nil and the given error code and message.
func WriteJSONError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Data:  nil,
		Error: &APIError{Code: code, Message: message},
	})
}
