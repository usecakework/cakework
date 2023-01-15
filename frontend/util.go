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

func sanitizeTaskName(task string) string {
	return strings.Replace(strings.ToLower(task), "_", "-", -1)
}
