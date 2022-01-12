package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger the logger struct
type Logger struct {
	logger    *log.Logger
	logLevel  level
	oldWriter io.Writer
}

// New init log
func New(o Options) *Logger {
	var w io.Writer = &lumberjack.Logger{
		Filename:   o.filename,
		MaxSize:    o.maxSize, // MB
		MaxBackups: o.maxBackups,
		MaxAge:     o.maxAge,
	}
	if len(o.writers) > 0 {
		o.writers = append(o.writers, w)
		w = io.MultiWriter(o.writers...)
	}

	l := log.New(w, "", log.Ldate|log.Ltime)
	return &Logger{
		logger:   l,
		logLevel: o.level,
	}
}

func (logger *Logger) Discard() {
	logger.oldWriter = logger.logger.Writer()
	logger.logger.SetOutput(io.Discard)
}

func (logger *Logger) ResumeWriter() {
	if logger.oldWriter != nil {
		logger.logger.SetOutput(logger.oldWriter)
		logger.oldWriter = nil
	}
}

// Log makes use of log
func (logger *Logger) Log(file string, line int, l level, v ...interface{}) {
	fl := fmt.Sprintf("[%5s] %s:%d:", levelString[l], file, line)
	lv := fmt.Sprint(v...)
	logger.logger.Println(fl, lv)
}

// Logf makes use of log
func (logger *Logger) Logf(file string, line int, l level, format string, v ...interface{}) {
	fl := fmt.Sprintf("[%5s] %s:%d: ", levelString[l], file, line)
	lv := fmt.Sprintf(format, v...)
	logger.logger.Print(fl, lv)
}

// WithLevel logs with the level specified
func (logger *Logger) WithLevel(file string, line int, l level, v ...interface{}) {
	if l > logger.logLevel {
		return
	}
	logger.Log(file, line, l, v...)
}

// WithLevelf logs with the level specified
func (logger *Logger) WithLevelf(file string, line int, l level, format string, v ...interface{}) {
	if l > logger.logLevel {
		return
	}
	logger.Logf(file, line, l, format, v...)
}

// Trace provides trace level logging
func (logger *Logger) Trace(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevel(filepath.Base(file), line, LevelTrace, v...)
}

// Tracef provides trace level logging
func (logger *Logger) Tracef(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevelf(filepath.Base(file), line, LevelTrace, format, v...)
}

// Debug provides debug level logging
func (logger *Logger) Debug(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevel(filepath.Base(file), line, LevelDebug, v...)
}

// Debugf provides debug level logging
func (logger *Logger) Debugf(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevelf(filepath.Base(file), line, LevelDebug, format, v...)
}

// Info provides info level logging
func (logger *Logger) Info(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevel(filepath.Base(file), line, LevelInfo, v...)
}

// Infof provides info level logging
func (logger *Logger) Infof(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevelf(filepath.Base(file), line, LevelInfo, format, v...)
}

// Warn provides warn level logging
func (logger *Logger) Warn(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevel(filepath.Base(file), line, LevelWarn, v...)
}

// Warnf provides warn level logging
func (logger *Logger) Warnf(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevelf(filepath.Base(file), line, LevelWarn, format, v...)
}

// Error provides error level logging
func (logger *Logger) Error(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevel(filepath.Base(file), line, LevelError, v...)
}

// Errorf provides error level logging
func (logger *Logger) Errorf(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevelf(filepath.Base(file), line, LevelError, format, v...)
}

// Fatal provides fatal level logging
func (logger *Logger) Fatal(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevel(filepath.Base(file), line, LevelFatal, v...)
	os.Exit(1)
}

// Fatalf provides fatal level logging
func (logger *Logger) Fatalf(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logger.WithLevelf(filepath.Base(file), line, LevelFatal, format, v...)
	os.Exit(1)
}

// Writer get log writer
func (logger *Logger) Writer() io.Writer {
	return logger.logger.Writer()
}
