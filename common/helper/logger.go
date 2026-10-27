package helper

import (
	"fmt"
	"time"
)

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	cyan   = "\033[36m"
)

func LogInfo(message string, args ...any) {
	log(cyan, "INFO", message, args...)
}

func LogTransaction(action, resource, message string, args ...any) {
	color := yellow
	switch action {
	case "CREATE":
		color = green
	case "DELETE", "ERROR":
		color = red
	}
	log(color, action, "%s", resource+": "+fmt.Sprintf(message, args...))
}

func log(color, level, message string, args ...any) {
	fmt.Printf("%s%s [%s] %s%s\n",
		color,
		time.Now().Format("2006-01-02 15:04:05"),
		level,
		fmt.Sprintf(message, args...),
		reset,
	)
}
