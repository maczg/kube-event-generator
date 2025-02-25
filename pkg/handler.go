package pkg

import "github.com/gin-gonic/gin"

type Handler struct {
	R *gin.Engine
	M *Manager
}

func NewHandler(r *gin.Engine, m *Manager) *Handler {
	return &Handler{R: r, M: m}
}

func (h *Handler) RegisterRoutes() {
	h.R.GET("/submit", h.submit)
	h.R.GET("/evict", h.evict)
}

func (h *Handler) submit(c *gin.Context) {
	// Implement the submit handler
}

func (h *Handler) evict(c *gin.Context) {
	// Implement the evict handler
}
