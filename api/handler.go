package api

import (
	"github.com/gin-gonic/gin"
	"github.com/maczg/kube-event-generator/internal/event"
	"github.com/maczg/kube-event-generator/pkg/factory"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	R *gin.Engine
	M *event.Manager
}

func NewHandler(m *event.Manager) *Handler {
	r := gin.Default()
	h := &Handler{R: r, M: m}
	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	h.R.GET("/submit", h.submit)
	h.R.GET("/evict", h.evict)
}

func (h *Handler) submit(c *gin.Context) {
	type Request struct {
		After    int  `json:"after"`
		Duration *int `json:"duration,omitempty"`
	}
	var req Request

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	p := factory.NewPod()
	h.M.ScheduleAt(p, &req.After, req.Duration)
	c.JSON(200, gin.H{"message": "event enqueued"})
	return
}

func (h *Handler) evict(c *gin.Context) {
	//
}

func (h *Handler) Run() {
	logrus.Fatalln(h.R.Run(":8080"))
}
