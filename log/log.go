// Package log 提供日志记录功能
//
// 对应 SlightPHP 的 SError 插件中的日志功能。
// 支持日志分级、文件和终端输出。
package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// 日志级别
// ---------------------------------------------------------------------------

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

func (l Level) String() string {
	if name, ok := levelNames[l]; ok {
		return name
	}
	return "UNKNOWN"
}

// ---------------------------------------------------------------------------
// Logger
// ---------------------------------------------------------------------------

// Logger 日志记录器
type Logger struct {
	mu       sync.Mutex
	level    Level
	logger   *log.Logger
	out      io.Writer
	file     *os.File
	filePath string
}

// Option 日志配置选项
type Option func(*Logger)

// WithLevel 设置日志级别
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithOutput 设置日志输出目标
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.out = w
		l.logger = log.New(w, "", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

// WithFile 设置日志文件输出
func WithFile(filePath string) Option {
	return func(l *Logger) {
		l.filePath = filePath
	}
}

// New 创建一个新的日志记录器
func New(opts ...Option) *Logger {
	l := &Logger{
		level: INFO,
		out:   os.Stdout,
	}

	l.logger = log.New(l.out, "", log.Ldate|log.Ltime|log.Lshortfile)

	for _, opt := range opts {
		opt(l)
	}

	// 如果指定了文件路径，打开文件
	if l.filePath != "" {
		f, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			l.file = f
			// 同时输出到文件和终端
			l.out = io.MultiWriter(os.Stdout, f)
			l.logger = log.New(l.out, "", log.Ldate|log.Ltime|log.Lshortfile)
		}
	}

	return l
}

// ---------------------------------------------------------------------------
// 日志方法
// ---------------------------------------------------------------------------

// log 输出日志
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	l.logger.Output(3, fmt.Sprintf("[%s] %s", level, msg))

	// FATAL 级别退出
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 输出调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 输出信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 输出错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal 输出致命错误日志并退出
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// ---------------------------------------------------------------------------
// 级别设置
// ---------------------------------------------------------------------------

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Level 返回当前日志级别
func (l *Logger) Level() Level {
	return l.level
}

// Close 关闭日志文件
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// ---------------------------------------------------------------------------
// 默认日志实例
// ---------------------------------------------------------------------------

var defaultLogger = New()

// SetDefaultLogger 设置全局默认日志记录器
func SetDefaultLogger(l *Logger) {
	defaultLogger = l
}

// DefaultLogger 返回默认日志记录器
func DefaultLogger() *Logger {
	return defaultLogger
}

// Debug 使用默认日志记录器输出调试日志
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info 使用默认日志记录器输出信息日志
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn 使用默认日志记录器输出警告日志
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error 使用默认日志记录器输出错误日志
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal 使用默认日志记录器输出致命错误日志
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// ---------------------------------------------------------------------------
// 格式化输出
// ---------------------------------------------------------------------------

// formatTime 格式化时间
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}
