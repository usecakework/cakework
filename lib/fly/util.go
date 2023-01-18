package fly

import "github.com/usecakework/cakework/lib/util"

func GetFlyAppName(userId string, appName string, taskName string) string {
	return util.SanitizeUserId(userId) + "-" + util.SanitizeAppName(appName) + "-" + util.SanitizeTaskName(taskName)
}