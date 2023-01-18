package main

import (
	"strings"
)

func sanitizeUserId(userId string) string {
	return strings.Replace(strings.ToLower(userId), "_", "-", -1)
}

func sanitizeAppName(app string) string {
	return strings.Replace(strings.ToLower(app), "_", "-", -1)
}

func sanitizeProjectName(project string) string {
	return strings.Replace(strings.ToLower(project), "_", "-", -1)
}

func sanitizeTaskName(task string) string {
	return strings.Replace(strings.ToLower(task), "_", "-", -1)
}

// should be sanitized for fly
func getFlyAppName(userId string, appName string, taskName string) string {
	return sanitizeUserId(userId) + "-" + sanitizeAppName(appName) + "-" + sanitizeTaskName(taskName)
}
