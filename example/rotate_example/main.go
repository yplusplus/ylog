package main

import (
	"github.com/yplusplus/ylog"
)

var log ylog.Logger

func init() {
	l, err := ylog.NewRotateLogger(".", ylog.TRACE)
	if err != nil {
		panic(err)
	}

	log = l
}

func main() {
	log.Trace("this is a trace log.")
	log.Debug("this is a debug log.")
	log.Warn("this is a warn log.")
	log.Error("this is a error log.")
	log.Info("this is a info log.")
}
