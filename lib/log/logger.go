package log

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

type logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
}

func NewLogger() Logger {
	return &logger{
		debugLogger: log.New(os.Stdout, "", 0),
		infoLogger:  log.New(os.Stdout, "", 0),
		warnLogger:  log.New(os.Stdout, "", 0),
		errorLogger: log.New(os.Stderr, "", 0),
		fatalLogger: log.New(os.Stderr, "", 0),
	}
}

func (l *logger) logWithCaller(level string, logger *log.Logger, format string, args ...any) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	prefix := fmt.Sprintf("%s %s: %s:%d: ", timestamp, level, file, line)
	logger.Printf(prefix+format, args...)
}

func logWithCallerGlobal(level string, logger *log.Logger, format string, args ...any) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	prefix := fmt.Sprintf("%s %s: %s:%d: ", timestamp, level, file, line)
	logger.Printf(prefix+format, args...)
}

func (l *logger) Debugf(format string, args ...any) {
	l.logWithCaller("DEBUG", l.debugLogger, format, args...)
}

func (l *logger) Infof(format string, args ...any) {
	l.logWithCaller("INFO", l.infoLogger, format, args...)
}

func (l *logger) Warnf(format string, args ...any) {
	l.logWithCaller("WARN", l.warnLogger, format, args...)
}

func (l *logger) Errorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	l.logWithCaller("ERROR", l.errorLogger, err.Error())
	return err
}

func (l *logger) Fatalf(format string, args ...any) {
	l.logWithCaller("FATAL", l.fatalLogger, format, args...)
	os.Exit(1)
}

var defaultLogger Logger = NewLogger()

func Debugf(format string, args ...any) {
	logWithCallerGlobal("DEBUG", defaultLogger.(*logger).debugLogger, format, args...)
}

func Infof(format string, args ...interface{}) {
	logWithCallerGlobal("INFO", defaultLogger.(*logger).infoLogger, format, args...)
}

func Warnf(format string, args ...interface{}) {
	logWithCallerGlobal("WARN", defaultLogger.(*logger).warnLogger, format, args...)
}

func Errorf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	logWithCallerGlobal("ERROR", defaultLogger.(*logger).errorLogger, err.Error())
	return err
}

func Fatalf(format string, args ...interface{}) {
	logWithCallerGlobal("FATAL", defaultLogger.(*logger).fatalLogger, format, args...)
	os.Exit(1)
}
