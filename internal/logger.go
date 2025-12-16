package internal

import (
	"fmt"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel LogLevel = INFO
	levelNames            = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
	// ANSI 颜色
	levelColors = map[LogLevel]string{
		DEBUG: "\033[36m", // 青色
		INFO:  "\033[32m", // 绿色
		WARN:  "\033[33m", // 黄色
		ERROR: "\033[31m", // 红色
	}
	resetColor = "\033[0m"
)

func InitLogger() {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug", "DEBUG":
		currentLevel = DEBUG
	case "warn", "WARN":
		currentLevel = WARN
	case "error", "ERROR":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}
}

func log(level LogLevel, format string, v ...interface{}) {
	if level < currentLevel {
		return
	}
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s[%s]%s %s %s\n", levelColors[level], levelNames[level], resetColor, timestamp, msg)
}

func LogDebug(format string, v ...interface{}) {
	log(DEBUG, format, v...)
}

func LogInfo(format string, v ...interface{}) {
	log(INFO, format, v...)
}

func LogWarn(format string, v ...interface{}) {
	log(WARN, format, v...)
}

func LogError(format string, v ...interface{}) {
	log(ERROR, format, v...)
}
