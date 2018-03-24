package main

import (
	"os"

	"github.com/yplusplus/ylog"
)

var log ylog.Logger

func init() {
	log = ylog.NewWriterLogger(os.Stderr, ylog.TRACE)
}

func main() {
	log.Trace("this is a trace log.")
	log.Debug("this is a debug log.")
	log.Warn("this is a warn log.")
	log.Error("this is a error log.")
	log.Info("this is a info log.")
}
