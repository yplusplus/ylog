package ylog

// log level
type LogLevel int32

// all log level
const (
	TRACE LogLevel = iota
	DEBUG
	WARN
	ERROR
	INFO
	FATAL
)

func (level LogLevel) LogLevelName() string {
	switch level {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case INFO:
		return "INFO"
	case FATAL:
		return "FATAL"
	}
	return "unknown"
}

var (
	LogLevelMap = map[string]LogLevel{
		"TRACE": TRACE,
		"DEBUG": DEBUG,
		"WARN":  WARN,
		"ERROR": ERROR,
		"INFO":  INFO,
		"FATAL": FATAL,
	}
)

type Logger interface {
	Tracef(format string, v ...interface{})
	Trace(v ...interface{})

	Debugf(format string, v ...interface{})
	Debug(v ...interface{})

	Warnf(format string, v ...interface{})
	Warn(v ...interface{})

	Errorf(format string, v ...interface{})
	Error(v ...interface{})

	Infof(format string, v ...interface{})
	Info(v ...interface{})

	Fatalf(format string, v ...interface{})
	Fatal(v ...interface{})
}
