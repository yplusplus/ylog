package ylog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WriterLogger outputs the log to an io.Writer
type WriterLogger struct {
	level LogLevel // log level

	mu    sync.Mutex // ensures atomic writes; protects the following fields
	buf   []byte     // buffer
	out   io.Writer  // destination for output
	flags int        // properties
}

func NewWriterLogger(out io.Writer, level LogLevel) *WriterLogger {
	return &WriterLogger{out: out, level: level, flags: LdefaultFlags}
}

// Output outputs content to log file
func (l *WriterLogger) Output(skipdepth int, s string) error {
	// get time early
	now := time.Now()

	var file string
	var line int
	var fn string
	pc, file, line, ok := runtime.Caller(skipdepth)
	if !ok {
		file = "????"
		line = 0
		fn = "unknown"
	} else {
		f := runtime.FuncForPC(pc)
		fn = f.Name()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.buf == nil || cap(l.buf) > DEFAULT_BUFFER_SIZE {
		l.buf = make([]byte, 0, DEFAULT_BUFFER_SIZE)
	} else {
		l.buf = l.buf[:0]
	}

	formatHeader(&l.buf, l.flags, now, file, line, fn)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}

	_, err := l.out.Write(l.buf)

	return err
}

// SetLogLevel sets log level for the logger
func (l *WriterLogger) SetLogLevel(level LogLevel) {
	atomic.StoreInt32((*int32)(&l.level), int32(level))
}

// LogLevel returns log level for the logger
func (l *WriterLogger) LogLevel() LogLevel {
	return LogLevel(atomic.LoadInt32((*int32)(&l.level)))
}

// Flags returns the flags for the logger
func (l *WriterLogger) Flags() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flags
}

// Flags sets the flags for the logger
func (l *WriterLogger) SetFlags(flags int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flags = flags
}

func (l *WriterLogger) Fatalf(format string, v ...interface{}) {
	l.Output(2, "FATAL|"+fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *WriterLogger) Fatal(v ...interface{}) {
	l.Output(2, "FATAL|"+fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *WriterLogger) Infof(format string, v ...interface{}) {
	l.Output(2, "INFO|"+fmt.Sprintf(format, v...))
}

func (l *WriterLogger) Info(v ...interface{}) {
	l.Output(2, "INFO|"+fmt.Sprintln(v...))
}

func (l *WriterLogger) Errorf(format string, v ...interface{}) {
	if l.LogLevel() <= ERROR {
		l.Output(2, "ERROR|"+fmt.Sprintf(format, v...))
	}
}

func (l *WriterLogger) Error(v ...interface{}) {
	if l.LogLevel() <= ERROR {
		l.Output(2, "ERROR|"+fmt.Sprintln(v...))
	}
}

func (l *WriterLogger) Warnf(format string, v ...interface{}) {
	if l.LogLevel() <= WARN {
		l.Output(2, "WARN|"+fmt.Sprintf(format, v...))
	}
}

func (l *WriterLogger) Warn(v ...interface{}) {
	if l.LogLevel() <= WARN {
		l.Output(2, "WARN|"+fmt.Sprintln(v...))
	}
}

func (l *WriterLogger) Tracef(format string, v ...interface{}) {
	if l.LogLevel() <= TRACE {
		l.Output(2, "TRACE|"+fmt.Sprintf(format, v...))
	}
}

func (l *WriterLogger) Trace(v ...interface{}) {
	if l.LogLevel() <= TRACE {
		l.Output(2, "TRACE|"+fmt.Sprintln(v...))
	}
}

func (l *WriterLogger) Debugf(format string, v ...interface{}) {
	if l.LogLevel() <= DEBUG {
		l.Output(2, "DEBUG|"+fmt.Sprintf(format, v...))
	}
}

func (l *WriterLogger) Debug(v ...interface{}) {
	if l.LogLevel() <= DEBUG {
		l.Output(2, "DEBUG|"+fmt.Sprintln(v...))
	}
}
