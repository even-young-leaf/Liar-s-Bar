package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	currentLevel = INFO
	logger       *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "", 0)

	// 从环境变量读取日志级别
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			currentLevel = DEBUG
		case "INFO":
			currentLevel = INFO
		case "WARN", "WARNING":
			currentLevel = WARN
		case "ERROR":
			currentLevel = ERROR
		}
	}
}

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (l Level) Color() string {
	switch l {
	case DEBUG:
		return "\033[36m" // Cyan
	case INFO:
		return "\033[32m" // Green
	case WARN:
		return "\033[33m" // Yellow
	case ERROR:
		return "\033[31m" // Red
	case FATAL:
		return "\033[35m" // Magenta
	default:
		return "\033[0m"
	}
}

func SetLevel(level Level) {
	currentLevel = level
}

func getCaller() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown:0"
	}
	// 只保留文件名，去掉完整路径
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func logf(level Level, format string, args ...interface{}) {
	if level < currentLevel {
		return
	}

	timestamp := time.Now().Format("2006/01/02 15:04:05.000")
	caller := getCaller()
	message := fmt.Sprintf(format, args...)

	colorReset := "\033[0m"
	levelColor := level.Color()

	output := fmt.Sprintf("%s %s[%s]%s %s - %s",
		timestamp,
		levelColor,
		level.String(),
		colorReset,
		caller,
		message,
	)

	logger.Println(output)

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 调试级别日志
func Debug(format string, args ...interface{}) {
	logf(DEBUG, format, args...)
}

// Info 信息级别日志
func Info(format string, args ...interface{}) {
	logf(INFO, format, args...)
}

// Warn 警告级别日志
func Warn(format string, args ...interface{}) {
	logf(WARN, format, args...)
}

// Error 错误级别日志
func Error(format string, args ...interface{}) {
	logf(ERROR, format, args...)
}

// Fatal 致命错误日志（会退出程序）
func Fatal(format string, args ...interface{}) {
	logf(FATAL, format, args...)
}

// 带上下文的日志记录器
type ContextLogger struct {
	context map[string]interface{}
}

func WithContext(context map[string]interface{}) *ContextLogger {
	return &ContextLogger{context: context}
}

func (cl *ContextLogger) formatContext() string {
	if len(cl.context) == 0 {
		return ""
	}

	parts := make([]string, 0, len(cl.context))
	for k, v := range cl.context {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return " [" + strings.Join(parts, ", ") + "]"
}

func (cl *ContextLogger) Debug(format string, args ...interface{}) {
	logf(DEBUG, fmt.Sprintf(format, args...)+cl.formatContext())
}

func (cl *ContextLogger) Info(format string, args ...interface{}) {
	logf(INFO, fmt.Sprintf(format, args...)+cl.formatContext())
}

func (cl *ContextLogger) Warn(format string, args ...interface{}) {
	logf(WARN, fmt.Sprintf(format, args...)+cl.formatContext())
}

func (cl *ContextLogger) Error(format string, args ...interface{}) {
	logf(ERROR, fmt.Sprintf(format, args...)+cl.formatContext())
}
