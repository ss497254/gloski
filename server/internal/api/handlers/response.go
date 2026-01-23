package handlers

// Re-export response functions from the response package for convenience
import (
	"net/http"

	"github.com/ss497254/gloski/internal/api/response"
)

// Response type alias
type Response = response.Response

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, status int, data interface{}) {
	response.JSON(w, status, data)
}

// Success writes a successful JSON response
func Success(w http.ResponseWriter, data interface{}) {
	response.Success(w, data)
}

// SuccessWithMessage writes a successful JSON response with a wrapper
func SuccessWithMessage(w http.ResponseWriter, data interface{}) {
	response.SuccessWithMessage(w, data)
}

// Error writes an error JSON response
func Error(w http.ResponseWriter, status int, message string) {
	response.Error(w, status, message)
}

// BadRequest writes a 400 error response
func BadRequest(w http.ResponseWriter, message string) {
	response.BadRequest(w, message)
}

// Unauthorized writes a 401 error response
func Unauthorized(w http.ResponseWriter, message string) {
	response.Unauthorized(w, message)
}

// Forbidden writes a 403 error response
func Forbidden(w http.ResponseWriter, message string) {
	response.Forbidden(w, message)
}

// NotFound writes a 404 error response
func NotFound(w http.ResponseWriter, message string) {
	response.NotFound(w, message)
}

// InternalError writes a 500 error response
func InternalError(w http.ResponseWriter, message string) {
	response.InternalError(w, message)
}
