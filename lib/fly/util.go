package fly

import (
	"strings"
)

func Sanitize(s string) string {
	return strings.Replace(strings.ToLower(s), "_", "-", -1)
}

func GetFlyAppName(userId string, appName string, taskName string) string {
	return Sanitize(userId) + "-" + Sanitize(appName) + "-" + Sanitize(taskName)
}