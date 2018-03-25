package ylog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DEFAULT_BUFFER_SIZE   = 4096              // default buffer size 4K, enough for most cases
	DEFAULT_LOG_FILE_SIZE = 512 * 1024 * 1024 // default log file size 512M
)

// RotateLogger will split Logs into several files according to log time and file size.
type RotateLogger struct {
	logDir       string   // log dir
	level        LogLevel // log level
	logSizeLimit int64    // log file size limit (KByte)

	mu     sync.Mutex // ensures atomic writes; protects the following fields
	flags  int        // properties
	buf    []byte     // buffer
	f      *os.File   // destination of output
	fname  string     // current log file name (format: YYYYMMDDHH.log[.ID])
	nbytes int64      // current log file size (Byte)
	fid    int32      // log file id
}

func NewRotateLogger(logDir string, level LogLevel) (*RotateLogger, error) {
	l := &RotateLogger{
		logDir:       logDir,
		level:        level,
		logSizeLimit: DEFAULT_LOG_FILE_SIZE,
		flags:        LdefaultFlags,
	}

	var err error
	// make log director
	if err = os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	now := time.Now()
	l.fname = getLogFileName(now, 0)
	l.fid = 0
	for i := 1; i < 100; i++ {
		filePath := filepath.Join(l.logDir, getLogFileName(now, int32(i)))
		_, err = os.Stat(filePath)
		if err == nil {
			l.fid++
		} else if os.IsNotExist(err) {
			break
		} else {
			return nil, err
		}
	}

	// create file
	err = l.createFile()
	if err != nil {
		return nil, err
	}

	return l, nil
}

func getLogFileName(t time.Time, id int32) string {
	fname := fmt.Sprintf("%04d%02d%02d%02d.log", t.Year(), t.Month(), t.Day(), t.Hour())
	if id > 0 {
		fname = fname + fmt.Sprintf(".%d", id)
	}
	return fname
}

// createFile creates a log file according to l.fileName and l.fid
func (l *RotateLogger) createFile() error {
	fileName := l.fname
	if l.fid > 0 {
		fileName += fmt.Sprintf(".%d", l.fid)
	}

	filePath := filepath.Join(l.logDir, fileName)
	var err error
	l.f, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// ignore error
	stat, err := l.f.Stat()
	if err == nil {
		l.nbytes = stat.Size()
	}

	return nil
}

func (l *RotateLogger) rotateFile(now time.Time) (err error) {
	needCreateFile := false

	currentFileName := getLogFileName(now, 0)
	if l.fname != currentFileName { // current log file is too old
		l.fname = currentFileName
		l.fid = 0
		needCreateFile = true
	} else if l.logSizeLimit > 0 && l.nbytes >= l.logSizeLimit { // current log file is too large
		l.fid++
		needCreateFile = true
	} else if l.f == nil {
		needCreateFile = true
	}

	if needCreateFile {
		if l.f != nil {
			l.f.Close()
			l.nbytes = 0
			l.f = nil
		}
		if err = l.createFile(); err != nil {
			// failed to create log file, we dont panic and try next output
			return
		}
	}
	return
}

// Flags returns the flags for the logger
func (l *RotateLogger) Flags() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flags
}

// Flags sets the flags for the logger
func (l *RotateLogger) SetFlags(flags int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flags = flags
}

// LogSizeLimit returns a single log file size limit
func (l *RotateLogger) LogSizeLimit() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logSizeLimit
}

// LogSizeLimit sets the single log file size limit for logger
// Give a non positive logSizeLimit to disable log splitting by size.
func (l *RotateLogger) SetLogSizeLimit(logSizeLimit int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logSizeLimit = logSizeLimit
}

// LogLevel returns the log level for the logger
func (l *RotateLogger) LogLevel() LogLevel {
	return LogLevel(atomic.LoadInt32((*int32)(&l.level)))
}

// SetLogLevel sets log level for the logger
func (l *RotateLogger) SetLogLevel(level LogLevel) {
	atomic.StoreInt32((*int32)(&l.level), int32(level))
}

// Output outputs content to log file
func (l *RotateLogger) Output(skipdepth int, s string) error {
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

	err := l.rotateFile(now)
	if err != nil {
		return err
	}

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

	nn, err := l.f.Write(l.buf)
	l.nbytes += int64(nn)

	return err
}

func (l *RotateLogger) Fatalf(format string, v ...interface{}) {
	l.Output(2, "FATAL|"+fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *RotateLogger) Fatal(v ...interface{}) {
	l.Output(2, "FATAL|"+fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *RotateLogger) Infof(format string, v ...interface{}) {
	l.Output(2, "INFO|"+fmt.Sprintf(format, v...))
}

func (l *RotateLogger) Info(v ...interface{}) {
	l.Output(2, "INFO|"+fmt.Sprintln(v...))
}

func (l *RotateLogger) Errorf(format string, v ...interface{}) {
	if l.LogLevel() <= ERROR {
		l.Output(2, "ERROR|"+fmt.Sprintf(format, v...))
	}
}

func (l *RotateLogger) Error(v ...interface{}) {
	if l.LogLevel() <= ERROR {
		l.Output(2, "ERROR|"+fmt.Sprintln(v...))
	}
}

func (l *RotateLogger) Warnf(format string, v ...interface{}) {
	if l.LogLevel() <= WARN {
		l.Output(2, "WARN|"+fmt.Sprintf(format, v...))
	}
}

func (l *RotateLogger) Warn(v ...interface{}) {
	if l.LogLevel() <= WARN {
		l.Output(2, "WARN|"+fmt.Sprintln(v...))
	}
}

func (l *RotateLogger) Tracef(format string, v ...interface{}) {
	if l.LogLevel() <= TRACE {
		l.Output(2, "TRACE|"+fmt.Sprintf(format, v...))
	}
}

func (l *RotateLogger) Trace(v ...interface{}) {
	if l.LogLevel() <= TRACE {
		l.Output(2, "TRACE|"+fmt.Sprintln(v...))
	}
}

func (l *RotateLogger) Debugf(format string, v ...interface{}) {
	if l.LogLevel() <= DEBUG {
		l.Output(2, "DEBUG|"+fmt.Sprintf(format, v...))
	}
}

func (l *RotateLogger) Debug(v ...interface{}) {
	if l.LogLevel() <= DEBUG {
		l.Output(2, "DEBUG|"+fmt.Sprintln(v...))
	}
}
