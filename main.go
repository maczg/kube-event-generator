package main

import (
	"github.com/maczg/kube-event-generator/api"
	"github.com/maczg/kube-event-generator/pkg"
	"github.com/sirupsen/logrus"
)

func init() {
	formatter := &logrus.TextFormatter{
		TimestampFormat: "15:04:05",
	}
	logrus.SetFormatter(formatter)
}

func main() {
	mgr, _ := pkg.NewManager()
	go mgr.Run()

	h := api.NewHandler(mgr)
	h.Run()
}
