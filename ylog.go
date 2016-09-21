// package ylog implements logging analogous to glog.
// It provides functions Debug, Trace, Warn, Error,
// plus formatting variants such as Debugf. Logs will split
// into several files according to log time and file size.
//
// Basic example:
//
//  func main() {
//      flag.Parse()            // parse flags
//      ylog.Init()             // initialize ylog system
//      defer ylog.Flush()      // ensure flush before exit
//
//      ylog.Debug("debug")
//      ylog.Trace("trace")
//      ylog.Warnf("Process failed: %s", err)
//  }
//
// Log output is buffered and written periodically using Flush. Programs
// should call Flush before exiting to guarantee all log output is written.
//
// -log-dir=""
//      Log files will be written to this directory instead of the default
//      temporary directory.
// -log-level=WARN
//      Log level is one of DEBUG, TRACE, WARN, ERROR and DEBUG < TRACE < WARN < ERROR.
//      Log below the preset log level will be ignored.
// -log-flush-period=5
//      period to call Flush(), default is 5s.
// -log-file-size=524288
//      split log file when reach specify log file size, default 512MB, 0 for not split.
package ylog

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogLevel int32

func (level *LogLevel) String() string {
	// default log level
	return "DEBUG"
}

func (level *LogLevel) Set(value string) (err error) {
	logLevel, ok := LogLevelMap[value]
	if !ok {
		err = fmt.Errorf("log level should be one of {debug, trace, warn, error}.")
		return
	}
	*level = logLevel
	return
}

// all log level
const (
	DEBUG LogLevel = iota
	TRACE
	WARN
	ERROR
)

var (
	LogLevelMap = map[string]LogLevel{
		"DEBUG": DEBUG,
		"TRACE": TRACE,
		"WARN":  WARN,
		"ERROR": ERROR,
	}
)

var logger = &loggerT{}

type loggerT struct {
	level       LogLevel      // log level
	logDir      string        // log dir
	logSize     int64         // log file size limit (KByte)
	flushPeriod int           // flush period (Second)
	mu          sync.Mutex    // ensures atomic writes; protects the following fields
	buf         []byte        // buffer
	out         *bufio.Writer // buf writer associated with f
	f           *os.File      // destination of output
	fname       string        // current log file name (format: YYYYMMDDHH.log)
	nbytes      int64         // current log file size (Byte)
	fid         int32         // log file id
}

func init() {
	flag.StringVar(&logger.logDir, "log-dir", os.TempDir(), "log dir")
	flag.Var(&logger.level, "log-level", "log level")
	flag.IntVar(&logger.flushPeriod, "log-flush-period", 5, "log flush period")
	flag.Int64Var(&logger.logSize, "log-file-size", 524288, "single log file max size (KB), default 512MB")
}

func Init() {

	if !flag.Parsed() {
		panic("flag should be parsed before calls ylog.Init()")
	}

	err := os.MkdirAll(logger.logDir, 0755)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	fname := fmt.Sprintf("%04d%02d%02d%02d.log", now.Year(), now.Month(), now.Day(), now.Hour())
	var logPath string
	for i := 0; ; i++ {
		logPath = fname
		if i > 0 {
			logPath = logPath + fmt.Sprintf(".%d", i)
		}
		logPath = filepath.Join(logger.logDir, logPath)
		_, err := os.Stat(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			panic(err)
		} else {
			logger.fid = int32(i)
		}
	}

	logger.fname = fname
	logPath = fname
	if logger.fid > 0 {
		logPath = logPath + fmt.Sprintf(".%d", logger.fid)
	}
	logPath = filepath.Join(logger.logDir, logPath)
	logger.f, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	logger.out = bufio.NewWriter(logger.f)
	stat, err := logger.f.Stat()
	if err == nil {
		logger.nbytes = stat.Size()
	} else {
		logger.nbytes = 0
	}

	go logger.flushDeamon()
}

func (l *loggerT) createFile(fname string) (err error) {
	if l.f != nil {
		l.out.Flush()
		l.f.Close()
	}

	var logPath string
	if l.fname != fname {
		logPath = filepath.Join(logger.logDir, fname)
		l.fid = 0
		l.fname = fname
	} else {
		l.fid++
		logPath = filepath.Join(logger.logDir, fname+fmt.Sprintf(".%d", l.fid))
	}
	l.f, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		l.out = nil
		return
	}
	l.out = bufio.NewWriter(l.f)

	stat, err := l.f.Stat() // ignore error
	if err != nil {
		l.nbytes = 0
	} else {
		l.nbytes = stat.Size()
	}

	return
}

func (l *loggerT) flushDeamon() {
	t := time.NewTicker(time.Duration(l.flushPeriod) * time.Second)
	for _ = range t.C {
		l.flush()
	}
}

// SetLogLevel sets log level
func (l *loggerT) SetLogLevel(level LogLevel) {
	atomic.StoreInt32((*int32)(&l.level), int32(level))
}

// LogLevel gets log level
func (l *loggerT) LogLevel() LogLevel {
	return LogLevel(atomic.LoadInt32((*int32)(&l.level)))
}

func (l *loggerT) flush() {
	l.mu.Lock()
	if l.f != nil {
		l.out.Flush()
	}
	l.mu.Unlock()
}

func (l *loggerT) Flush() {
	l.flush()
}

// formatHeader formats log prefix likes YYYYMMDD HH:MM:SS.NNNNNN|FILE:LINE|FUNC|
func (l *loggerT) formatHeader(t time.Time, file string, line int, fn string) {
	// set date and time
	l.buf = append(l.buf, fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())...)
	l.buf = append(l.buf,
		fmt.Sprintf(" %02d:%02d:%02d.%06d|", t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000)...)

	// set file, line and func name
	offset := strings.LastIndexByte(file, '/')
	l.buf = append(l.buf, file[offset+1:]...)
	l.buf = append(l.buf, ':')
	l.buf = append(l.buf, fmt.Sprintf("%d", line)...)
	l.buf = append(l.buf, '|')
	l.buf = append(l.buf, fn...)
	l.buf = append(l.buf, '|')
}

// Output outputs content to log
func (l *loggerT) Output(skipdepth int, s string) error {
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

	now := time.Now()
	l.buf = l.buf[:0]
	l.formatHeader(now, file, line, fn)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}

	// check if need to create new log file
	fname := fmt.Sprintf("%04d%02d%02d%02d.log", now.Year(), now.Month(), now.Day(), now.Hour())
	if l.f == nil || l.fname != fname || (l.logSize > 0 && (l.nbytes+int64(len(l.buf)))/1024 > l.logSize) {
		if err := l.createFile(fname); err != nil {
			// return err and try to create file again in next log??
			//panic(err)
			return err
		}
	}

	_, err := l.out.Write(l.buf)
	l.nbytes += int64(len(l.buf))

	return err
}

func (l *loggerT) Fatalf(format string, v ...interface{}) {
	l.Output(3, "FATAL|"+fmt.Sprintf(format, v...))
	l.Flush()
	os.Exit(1)
}

func (l *loggerT) Fatal(v ...interface{}) {
	l.Output(3, "FATAL|"+fmt.Sprintln(v...))
	l.Flush()
	os.Exit(1)
}

func (l *loggerT) Infof(format string, v ...interface{}) {
	l.Output(3, "INFO |"+fmt.Sprintf(format, v...))
}

func (l *loggerT) Info(v ...interface{}) {
	l.Output(3, "INFO |"+fmt.Sprintln(v...))
}

func (l *loggerT) Errorf(format string, v ...interface{}) {
	if l.LogLevel() <= ERROR {
		l.Output(3, "ERROR|"+fmt.Sprintf(format, v...))
	}
}

func (l *loggerT) Error(v ...interface{}) {
	if l.LogLevel() <= ERROR {
		l.Output(3, "ERROR|"+fmt.Sprintln(v...))
	}
}

func (l *loggerT) Warnf(format string, v ...interface{}) {
	if l.LogLevel() <= WARN {
		l.Output(3, "WARN |"+fmt.Sprintf(format, v...))
	}
}

func (l *loggerT) Warn(v ...interface{}) {
	if l.LogLevel() <= WARN {
		l.Output(3, "WARN |"+fmt.Sprintln(v...))
	}
}

func (l *loggerT) Tracef(format string, v ...interface{}) {
	if l.LogLevel() <= TRACE {
		l.Output(3, "TRACE|"+fmt.Sprintf(format, v...))
	}
}

func (l *loggerT) Trace(v ...interface{}) {
	if l.LogLevel() <= TRACE {
		l.Output(3, "TRACE|"+fmt.Sprintln(v...))
	}
}

func (l *loggerT) Debugf(format string, v ...interface{}) {
	if l.LogLevel() <= DEBUG {
		l.Output(3, "DEBUG|"+fmt.Sprintf(format, v...))
	}
}

func (l *loggerT) Debug(v ...interface{}) {
	if l.LogLevel() <= DEBUG {
		l.Output(3, "DEBUG|"+fmt.Sprintln(v...))
	}
}

func Flush() {
	logger.Flush()
}

func Output(skipdepth int, s string) error {
	return logger.Output(skipdepth+2, s)
}

func Fatalf(format string, v ...interface{}) {
	logger.Fatalf(format, v...)
}

func Fatal(v ...interface{}) {
	logger.Fatal(v...)
}

func Errorf(format string, v ...interface{}) {
	logger.Errorf(format, v...)
}

func Error(v ...interface{}) {
	logger.Error(v...)
}

func Warnf(format string, v ...interface{}) {
	logger.Warnf(format, v...)
}

func Warn(v ...interface{}) {
	logger.Warn(v...)
}

func Infof(format string, v ...interface{}) {
	logger.Infof(format, v...)
}

func Info(v ...interface{}) {
	logger.Info(v...)
}

func Tracef(format string, v ...interface{}) {
	logger.Tracef(format, v...)
}

func Trace(v ...interface{}) {
	logger.Trace(v...)
}

func Debugf(format string, v ...interface{}) {
	logger.Debugf(format, v...)
}

func Debug(v ...interface{}) {
	logger.Debug(v...)
}
