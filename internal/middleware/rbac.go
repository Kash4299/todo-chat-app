package middleware

import (
	"errors"
	"net/http"
	"slices"

	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type workspaceRoleGetter interface {
	GetRole(workspaceID, userID uuid.UUID) (string, error)
}

type RBACMiddleware struct {
	workspaceService workspaceRoleGetter
}

func NewRBACMiddleware(workspaceService service.IWorkspaceMemberService) *RBACMiddleware {
	return &RBACMiddleware{
		workspaceService: workspaceService,
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
		if err != nil || workspaceIDUUID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid workspaceID"})
			return
		}

		role, err := m.workspaceService.GetRole(workspaceIDUUID, userIDUUID)
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

		if !slices.Contains(requiredRoles, role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		c.Next()
	}
}
