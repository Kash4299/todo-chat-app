package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	health service.IHealthService
}

func NewHealthHandler(health service.IHealthService) *HealthHandler {
	return &HealthHandler{health: health}
}

// Healthz is the liveness probe: it reports only that the process is up and the
// HTTP server can respond. It must NOT touch external dependencies — putting a
// shared dependency (DB) here would make a transient blip restart every
// instance at once.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz is the readiness probe: it reports whether this instance can serve a
// request right now, by checking the dependencies required to serve (Postgres).
// On failure it returns 503 so the load balancer stops routing traffic — the
// pod is left alive to recover.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.health.Ready(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
