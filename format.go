package ylog

import (
	"fmt"
	"strings"
	"time"
)

// formatHeader formats log prefix likes YYYYMMDD HH:MM:SS.NNNNNN|FILE:LINE|FUNC|
func formatHeader(buf *[]byte, t time.Time, file string, line int, fn string) {
	// set date and time
	*buf = append(*buf, fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())...)
	*buf = append(*buf, fmt.Sprintf(" %02d:%02d:%02d.%06d|", t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000)...)

	// set file, line and func name
	offset := strings.LastIndexByte(file, '/')
	*buf = append(*buf, file[offset+1:]...)
	*buf = append(*buf, ':')
	*buf = append(*buf, fmt.Sprintf("%d", line)...)
	*buf = append(*buf, '|')
	*buf = append(*buf, fn...)
	*buf = append(*buf, '|')
}
