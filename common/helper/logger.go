package helper

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	cyan   = "\033[36m"
)

var appStartTime = time.Now()

func elapsed() string {
	secs := int(time.Since(appStartTime).Seconds())
	if secs > 9999 {
		secs = 9999
	}
	return fmt.Sprintf("[%04d]", secs)
}

func logFormatted(color, level, message, fields string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	tag := fmt.Sprintf("%s%s%s%s", color, level, elapsed(), reset)

	if fields == "" {
		fields = fmt.Sprintf("at=%q", now)
	}

	fmt.Printf("%s %-44s %s\n", tag, message, fields)
}

func LogInfo(message string, args ...any) {
	logFormatted(cyan, "INFO", fmt.Sprintf(message, args...), "")
}

func LogWarn(message string, args ...any) {
	logFormatted(yellow, "WARN", fmt.Sprintf(message, args...), "")
}

func LogError(message string, args ...any) {
	logFormatted(red, "ERRO", fmt.Sprintf(message, args...), "")
}

func LogTransaction(action, resource, message string, args ...any) {
	color := cyan
	level := "INFO"
	switch action {
	case "CREATE":
		color = green
	case "DELETE":
		color = yellow
	case "ERROR":
		color = red
		level = "ERRO"
	}
	msg := fmt.Sprintf("[%s] %s: %s", action, resource, fmt.Sprintf(message, args...))
	logFormatted(color, level, msg, "")
}

func LogIncomingRequest(r *http.Request) {
	now := time.Now().Format("2006-01-02 15:04:05")
	ip := clientIP(r)
	uri := r.URL.RequestURI()
	if uri == "" {
		uri = r.URL.Path
	}
	fields := fmt.Sprintf("at=%q ip=%q method=%s uri=%s", now, ip, r.Method, uri)
	logFormatted(cyan, "INFO", "incoming request", fields)
}

func LogHTTPError(r *http.Request, status int) {
	now := time.Now().Format("2006-01-02 15:04:05")
	ip := clientIP(r)
	uri := r.URL.RequestURI()
	if uri == "" {
		uri = r.URL.Path
	}
	statusText := http.StatusText(status)
	if statusText == "" {
		statusText = "Unknown Error"
	}
	msg := fmt.Sprintf("http error %d - %s", status, statusText)
	fields := fmt.Sprintf("at=%q ip=%q method=%s uri=%s", now, ip, r.Method, uri)
	logFormatted(red, "ERRO", msg, fields)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "127.0.0.1"
}
