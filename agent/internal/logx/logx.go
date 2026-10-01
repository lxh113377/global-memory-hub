// Package logx 极简分级日志: 追加写 ~/.fenjue/logs/agent.log, WARN 以上同步输出 stderr。
package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level 日志级别。
type Level int

// 级别常量。
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	mu sync.Mutex
	f  *os.File
)

func levelName(l Level) string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	default:
		return "ERROR"
	}
}

// Init 打开日志文件(追加)。失败时仅 stderr 提示, 不中断程序。
func Init(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "logx: mkdir %s: %v\n", filepath.Dir(path), err)
		return
	}
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logx: open %s: %v\n", path, err)
		return
	}
	mu.Lock()
	f = fp
	mu.Unlock()
}

// Close 关闭日志文件。
func Close() {
	mu.Lock()
	if f != nil {
		f.Close()
		f = nil
	}
	mu.Unlock()
}

func logf(l Level, format string, args ...any) {
	line := time.Now().Format("2006-01-02 15:04:05") + " " + levelName(l) + " " + fmt.Sprintf(format, args...) + "\n"
	mu.Lock()
	defer mu.Unlock()
	if f != nil {
		f.WriteString(line)
	}
	if l >= LevelWarn {
		os.Stderr.WriteString(line)
	}
}

// Debug 记录调试信息。
func Debug(format string, args ...any) { logf(LevelDebug, format, args...) }

// Info 记录常规信息。
func Info(format string, args ...any) { logf(LevelInfo, format, args...) }

// Warn 记录警告。
func Warn(format string, args ...any) { logf(LevelWarn, format, args...) }

// Error 记录错误。
func Error(format string, args ...any) { logf(LevelError, format, args...) }
