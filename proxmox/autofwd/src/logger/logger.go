package logger

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type LogApp struct {
	mu          sync.Mutex
	fileTrack   map[string]io.Writer
	multiWriter []io.Writer
	prefix      string
	*log.Logger
}

var Logger *LogApp

// Return a new *LogApp, if f is nil return logger to os.Stdout.
func New(f *os.File, console bool) *LogApp {
	initLogApp := &LogApp{prefix: prefixNormal}
	if console {
		initLogApp.AddNewFileWriter(os.Stdout)
	}
	if f != nil {
		initLogApp.AddNewFileWriter(f)
	}
	return initLogApp
}

func InitWithFile(f *os.File, console bool) error {
	if f != nil {
		if _, err := f.Stat(); errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	Logger = &LogApp{prefix: prefixNormal}
	if console {
		Logger.AddNewFileWriter(os.Stdout)
	}
	if f != nil {
		Logger.AddNewFileWriter(f)
	}
	return nil
}

// Add a new io.Writer to the logger so now you can log on different file.
func (l *LogApp) AddNewFileWriter(f *os.File) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.prefix == "" {
		l.prefix = prefixNormal
	}
	if l.fileTrack == nil {
		l.fileTrack = make(map[string]io.Writer)
	}
	l.fileTrack[f.Name()] = f
	l.multiWriter = append(l.multiWriter, f)
	l.Logger = log.New(io.MultiWriter(l.multiWriter...), "", 0)
}

// Remove a io.Writer from the logger so you dont write to that file anymore,
// return error if *os.File wasn't found.
func (l *LogApp) RemoveFileWriter(f *os.File) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.fileTrack[f.Name()]
	if ok {
		for i, writer := range l.multiWriter {
			if writer == w {
				l.multiWriter = append(l.multiWriter[:i], l.multiWriter[i+1:]...)
				l.Logger = log.New(io.MultiWriter(l.multiWriter...), "", 0)
				f.Close()
				return nil
			}
		}
	}
	return errors.New("could not return file")
}

// Call SetPrefix for logger.
func (l *LogApp) SetPrefix(prefix string) {
	l.prefix = prefix
}

// Call SetPrefix for logger.
func SetPrefix(prefix string) {
	Logger.SetPrefix(prefix)
}

// Use Normal Prefix do a log.Printf.
func (l *LogApp) Printf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixNormal)
	l.customPrintf(format, v...)
}

// Use Normal Prefix do a log.Printf.
func Printf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixNormal)
	Logger.customPrintf(format, v...)
}

// Use Error Prefix do a log.Printf.
func (l *LogApp) Errf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixError)
	l.customPrintf(format, v...)
}

// Use Error Prefix do a log.Printf.
func Errf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixError)
	Logger.customPrintf(format, v...)
}

// Use Debug Prefix do a log.Printf.
func (l *LogApp) Debugf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixDebug)
	l.customPrintf(format, v...)
}

// Use Debug Prefix do a log.Printf.
func Debugf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixDebug)
	Logger.customPrintf(format, v...)
}

// Use Error Prefix do a log.Printf and os.Exit(1).
func (l *LogApp) Fatalf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixError)
	l.customPrintf(format, v...)
	os.Exit(1)
}

// Use Error Prefix do a log.Printf and os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixError)
	Logger.customPrintf(format, v...)
	os.Exit(1)
}

// Use Daemon Prefix do a log.Printf
func (l *LogApp) DaemonF(format string, v ...interface{}) {
	Logger.SetPrefix(prefixDaemon)
	l.customPrintf(format, v...)
}

// Use Daemon Prefix do a log.Printf
func Daemonf(format string, v ...interface{}) {
	Logger.SetPrefix(prefixDaemon)
	Logger.customPrintf(format, v...)
}

// custom Printf for better logger output.
func (l *LogApp) customPrintf(format string, v ...interface{}) {
	currentTime := time.Now().Format(time.RFC3339Nano)
	stackFrameSkip := 2
	_, file, line, ok := runtime.Caller(stackFrameSkip)
	if !ok {
		l.Logger.Printf("%v [ %s ]: %s", currentTime, l.prefix, fmt.Sprintf(format, v...))
	} else {
		l.Logger.Printf("%v [ %s ] %s:%d: %s", currentTime, l.prefix, filepath.Base(file), line, fmt.Sprintf(format, v...))
	}
}
