package util

import (
	"strings"
)

func SanitizeUserId(userId string) string {
	return strings.Replace(strings.ToLower(userId), "_", "-", -1)
}

func SanitizeAppName(app string) string {
	return strings.Replace(strings.ToLower(app), "_", "-", -1)
}

func SanitizeProjectName(project string) string {
	return strings.Replace(strings.ToLower(project), "_", "-", -1)
}

func SanitizeTaskName(task string) string {
	return strings.Replace(strings.ToLower(task), "_", "-", -1)
}