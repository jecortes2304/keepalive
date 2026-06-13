package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *log.Logger
	file     *os.File
	mu       sync.Mutex
)

func Init(logPath string) error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("creating log dir: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	file = f
	instance = log.New(f, "", 0)
	return nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		file.Close()
		file = nil
	}
}

func Info(format string, args ...any) {
	write("INFO", format, args...)
}

func Error(format string, args ...any) {
	write("ERROR", format, args...)
}

func Warn(format string, args ...any) {
	write("WARN", format, args...)
}

func write(level, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if instance == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	instance.Printf("%s [%s] %s", time.Now().Format("2006-01-02 15:04:05"), level, msg)
}
