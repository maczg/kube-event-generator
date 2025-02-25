package api

import (
	"github.com/gin-gonic/gin"
	"github.com/maczg/kube-event-generator/pkg"
	"github.com/sirupsen/logrus"
	"time"
)

type Handler struct {
	R *gin.Engine
	M *pkg.Manager
}

func NewHandler(m *pkg.Manager) *Handler {
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
		After    int    `json:"after"`
		Duration *int   `json:"duration,omitempty"`
		Type     string `json:"type"`
	}
	var req Request

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	at := time.Now().Add(time.Duration(req.After) * time.Second)
	var d time.Duration
	if req.Duration != nil {
		d = time.Duration(*req.Duration) * time.Second
	}
	e := pkg.NewEvent(&at, &d)
	h.M.EnqueueEvent(e)
	c.JSON(200, gin.H{"message": "event enqueued"})
	return
}

func (h *Handler) evict(c *gin.Context) {
	//
}

func (h *Handler) Run() {
	logrus.Fatalln(h.R.Run(":8080"))
}
