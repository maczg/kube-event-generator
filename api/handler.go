package api

import (
	"github.com/gin-gonic/gin"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	R *gin.Engine
	S *scenario.Scheduler
}

func NewHandler(s *scenario.Scheduler) *Handler {
	r := gin.Default()
	h := &Handler{R: r, S: s}
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
	var req scenario.Event

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	h.S.AddEvent(&req)
	c.JSON(200, gin.H{"message": "event enqueued"})
	return
}

func (h *Handler) evict(c *gin.Context) {
	//
}

func (h *Handler) Run() {
	logrus.Fatalln(h.R.Run(":8080"))
}
