package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ===========================================================================
// Response shapes
//
// Success:  { "data": <T> }
// List:     { "data": [...], "meta": { "total": N, "cursor": "..." } }
// Error:    { "error": { "code": "SNAKE_CASE", "message": "human text" } }
//
// HTTP status codes carry the numeric status — we do not repeat them in body.
// "code" in error body is a machine-readable string for frontend i18n mapping.
// ===========================================================================

// ErrorCode is a machine-readable string clients use to map to UI messages.
type ErrorCode string

const (
	// Generic HTTP-level codes
	CodeInvalidInput  ErrorCode = "INVALID_INPUT"
	CodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	CodeForbidden     ErrorCode = "FORBIDDEN"
	CodeNotFound      ErrorCode = "NOT_FOUND"
	CodeConflict      ErrorCode = "CONFLICT"
	CodeInternalError ErrorCode = "INTERNAL_ERROR"

	// Auth domain
	CodeEmailTaken         ErrorCode = "EMAIL_TAKEN"
	CodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	CodePasswordTooShort   ErrorCode = "PASSWORD_TOO_SHORT"
	CodePasswordAlreadySet ErrorCode = "PASSWORD_ALREADY_SET"
	CodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	CodeTokenInvalid       ErrorCode = "TOKEN_INVALID"
	CodeEmailNotVerified   ErrorCode = "EMAIL_NOT_VERIFIED"

	// Workspace domain
	CodeWorkspaceNotFound ErrorCode = "WORKSPACE_NOT_FOUND"

	// Invitation domain
	CodeInvitationExpired ErrorCode = "INVITATION_EXPIRED"
	CodeInvitationUsed    ErrorCode = "INVITATION_USED"
	CodeAlreadyMember     ErrorCode = "ALREADY_MEMBER"
)

// PaginationMeta is included in list responses.
// Page-based: use for bounded resource lists (workspaces, tasks, members).
// Cursor-based (future): use for unbounded infinite-scroll feeds (messages).
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// --- internal shapes (unexported — callers use helper functions) ---

type dataResponse struct {
	Data any `json:"data"`
}

type listResponse struct {
	Data any             `json:"data"`
	Meta *PaginationMeta `json:"meta,omitempty"`
}

type errorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

// ===========================================================================
// Success helpers
// ===========================================================================

// OK writes HTTP 200 with a data envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, dataResponse{Data: data})
}

// Created writes HTTP 201 with a data envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, dataResponse{Data: data})
}

// NoContent writes HTTP 204 with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// List writes HTTP 200 with a data array and optional pagination metadata.
func List(c *gin.Context, data any, meta *PaginationMeta) {
	c.JSON(http.StatusOK, listResponse{Data: data, Meta: meta})
}

// ===========================================================================
// Error helpers
// ===========================================================================

func BadRequest(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusBadRequest, errorResponse{Error: errorDetail{code, message}})
}

func Unauthorized(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusUnauthorized, errorResponse{Error: errorDetail{code, message}})
}

func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, errorResponse{
		Error: errorDetail{CodeForbidden, "you do not have permission to perform this action"},
	})
}

func ForbiddenCode(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusForbidden, errorResponse{Error: errorDetail{code, message}})
}

func NotFound(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusNotFound, errorResponse{Error: errorDetail{code, message}})
}

func Conflict(c *gin.Context, code ErrorCode, message string) {
	c.JSON(http.StatusConflict, errorResponse{Error: errorDetail{code, message}})
}

func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, errorResponse{
		Error: errorDetail{CodeInternalError, "an unexpected error occurred"},
	})
}
