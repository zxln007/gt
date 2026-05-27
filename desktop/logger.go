package main

import (
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// WailsLogWriter 实现 io.Writer，用于拦截 GT 内核日志并实时推送到前端
type WailsLogWriter struct {
	mu     sync.Mutex
	buffer []string
	maxLen int
}

// NewWailsLogWriter 创建一个日志拦截器
func NewWailsLogWriter() *WailsLogWriter {
	return &WailsLogWriter{
		buffer: make([]string, 0, 1000),
		maxLen: 1000,
	}
}

// Write 实现 io.Writer 接口
func (w *WailsLogWriter) Write(p []byte) (n int, err error) {
	w.AppendLine(string(p))
	return len(p), nil
}

func (w *WailsLogWriter) AppendLine(line string) {
	logLine := strings.TrimRight(line, "\r\n")
	if logLine == "" {
		return
	}

	w.mu.Lock()
	if len(w.buffer) >= w.maxLen {
		w.buffer = w.buffer[1:]
	}
	w.buffer = append(w.buffer, logLine)
	w.mu.Unlock()

	// 实时通过 Wails v3 事件管道推送给前端
	app := application.Get()
	if app != nil {
		app.Event.EmitEvent(&application.CustomEvent{
			Name: "gt:log",
			Data: logLine,
		})
	}
}

// GetLogs 获取当前缓存的日志
func (w *WailsLogWriter) GetLogs() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	logsCopy := make([]string, len(w.buffer))
	copy(logsCopy, w.buffer)
	return logsCopy
}

// Clear 清空日志缓存
func (w *WailsLogWriter) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buffer = w.buffer[:0]
}
