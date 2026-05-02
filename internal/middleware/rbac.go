package middleware

import (
	"errors"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IWorkspaceMemberRoleService interface {
	GetRole(workspaceID, userID uuid.UUID) (string, error)
}

type RBACMiddleware struct {
	workspaceMemberService IWorkspaceMemberRoleService
}

func NewRBACMiddleware(workspaceMemberService IWorkspaceMemberRoleService) *RBACMiddleware {
	return &RBACMiddleware{
		workspaceMemberService: workspaceMemberService,
	}
}

func (m *RBACMiddleware) RequireWorkspaceRole(requiredRoles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(UserIDContextKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		userIDUUID, ok := userID.(uuid.UUID)
		if !ok || userIDUUID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		workspaceID := c.Param("workspaceID")
		if workspaceID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "workspaceID is required"})
			return
		}

		workspaceIDUUID, err := uuid.Parse(workspaceID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid workspaceID"})
			return
		}

		role, err := m.workspaceMemberService.GetRole(workspaceIDUUID, userIDUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
				return
			}

			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		hasRole := slices.Contains(requiredRoles, role)
		if !hasRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		c.Next()
	}
}
