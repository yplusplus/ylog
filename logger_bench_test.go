package ylog

import (
	"log"
	"os"
	"testing"
)

func BenchmarkGolangLogger(b *testing.B) {
	nullf, err := os.OpenFile("/dev/null", os.O_WRONLY, 0666)
	if err != nil {
		b.Fatal("%v", err)
	}
	defer nullf.Close()
	logger := log.New(nullf, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	logger.SetPrefix("DEBUG|")
	for i := 0; i < b.N; i++ {
		logger.Println("testing")
	}
}

func BenchmarkGolangLoggerParallel(b *testing.B) {
	nullf, err := os.OpenFile("/dev/null", os.O_WRONLY, 0666)
	if err != nil {
		b.Fatal("%v", err)
	}
	defer nullf.Close()
	logger := log.New(nullf, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	logger.SetPrefix("DEBUG|")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Println("testing")
		}
	})
}

func BenchmarkWriterLogger(b *testing.B) {
	nullf, err := os.OpenFile("/dev/null", os.O_WRONLY, 0666)
	if err != nil {
		b.Fatal("%v", err)
	}
	defer nullf.Close()
	logger := NewWriterLogger(nullf, TRACE)
	logger.SetFlags(logger.Flags() & (^Lloglevel))
	for i := 0; i < b.N; i++ {
		logger.Debug("testing")
	}
}

func BenchmarkWriterLoggerParallel(b *testing.B) {
	nullf, err := os.OpenFile("/dev/null", os.O_WRONLY, 0666)
	if err != nil {
		b.Fatal("%v", err)
	}
	defer nullf.Close()
	logger := NewWriterLogger(nullf, TRACE)
	logger.SetFlags(logger.Flags() & (^Lloglevel))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Debug("testing")
		}
	})
}
