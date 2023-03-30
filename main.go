package main

import (
	log "github.com/sirupsen/logrus"
)

var Options struct {
}

func main() {
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})
}
