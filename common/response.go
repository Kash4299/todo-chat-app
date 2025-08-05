package common

import (
	"net/http"
)

// Response represents the standard API response structure
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    any    `json:"data"`
}

// SuccessResponse creates a success response with the given data
func SuccessResponse(data any) Response {
	return Response{
		Status:  "success",
		Message: "success",
		Code:    http.StatusOK,
		Data:    data,
	}
}

// SuccessResponseWithMessage creates a success response with custom message and data
func SuccessResponseWithMessage(message string, data interface{}) Response {
	return Response{
		Status:  "success",
		Message: message,
		Code:    http.StatusOK,
		Data:    data,
	}
}

// ErrorResponse creates an error response with the given code and message
func ErrorResponse(code int, message string) Response {
	return Response{
		Status:  "error",
		Message: message,
		Code:    code,
		Data:    nil,
	}
}

// BadRequestResponse creates a 400 Bad Request response
func BadRequestResponse(message string) Response {
	return ErrorResponse(http.StatusBadRequest, message)
}

// UnauthorizedResponse creates a 401 Unauthorized response
func UnauthorizedResponse(message string) Response {
	return ErrorResponse(http.StatusUnauthorized, message)
}

// ForbiddenResponse creates a 403 Forbidden response
func ForbiddenResponse(message string) Response {
	return ErrorResponse(http.StatusForbidden, message)
}

// NotFoundResponse creates a 404 Not Found response
func NotFoundResponse(message string) Response {
	return ErrorResponse(http.StatusNotFound, message)
}

// InternalServerErrorResponse creates a 500 Internal Server Error response
func InternalServerErrorResponse(message string) Response {
	return ErrorResponse(http.StatusInternalServerError, message)
}

// ValidationErrorResponse creates a 422 Unprocessable Entity response for validation errors
func ValidationErrorResponse(message string) Response {
	return ErrorResponse(http.StatusUnprocessableEntity, message)
}
