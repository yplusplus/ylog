package ylog

import "testing"

func TestSetLogLevel(t *testing.T) {
	logger.SetLogLevel(DEBUG)
	if DEBUG != logger.LogLevel() {
		t.Error("SetLogLevel test failed")
	}
	logger.SetLogLevel(TRACE)
	if TRACE != logger.LogLevel() {
		t.Error("SetLogLevel test failed")
	}
	logger.SetLogLevel(WARN)
	if WARN != logger.LogLevel() {
		t.Error("SetLogLevel test failed")
	}
	logger.SetLogLevel(ERROR)
	if ERROR != logger.LogLevel() {
		t.Error("SetLogLevel test failed")
	}
	t.Log("SetLogLevel test succ!")
}
