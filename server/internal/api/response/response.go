package response

import (
	"encoding/json"
	"net/http"

	"github.com/ss497254/gloski/internal/logger"
)

// Config holds response configuration
var config struct {
	DetailedErrors bool
}

// SetDetailedErrors configures whether detailed error messages are included in responses
func SetDetailedErrors(enabled bool) {
	config.DetailedErrors = enabled
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Success writes a successful JSON response
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

// SuccessWithMessage writes a successful JSON response with a wrapper
func SuccessWithMessage(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{Success: true, Data: data})
}

// Error writes an error JSON response
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Response{Success: false, Error: message})
}

// BadRequest writes a 400 error response
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// Unauthorized writes a 401 error response
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden writes a 403 error response
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message)
}

// NotFound writes a 404 error response
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// InternalError writes a 500 error response with context
// Always logs the detailed error, but only includes it in response if DetailedErrors is enabled
func InternalError(w http.ResponseWriter, context string, detail string) {
	logger.Error("%s: %s", context, detail)

	message := context
	if config.DetailedErrors && detail != "" {
		message = context + ": " + detail
	}

	Error(w, http.StatusInternalServerError, message)
}
