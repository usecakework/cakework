package fly

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/usecakework/cakework/lib/fly/api"
)

func Sanitize(s string) string {
	return strings.Replace(strings.ToLower(s), "_", "-", -1)
}

func GetFlyAppName(userId string, appName string, taskName string) string {
	return Sanitize(userId) + "-" + Sanitize(appName) + "-" + Sanitize(taskName)
}

func GetLatestImage(flyApp string, db *sql.DB) (string, error) {
	var config api.MachineConfig
	err := db.QueryRow("SELECT image, machineId FROM FlyMachine WHERE flyApp = ? ORDER BY createdAt DESC LIMIT 1", flyApp).Scan(&config.Config.Image, &config.MachineId)
	if err != nil {
		return "", err
	}

	if config.Config.Image != "" {
		return config.Config.Image, nil
	} else {
		return "", errors.New("Got the latest deployed FlyMachine, but image is null")
	}
}

// returns true if we've successfully previously built an image for this app which can then be 
func ImageExists(userId string, project string, task string, db *sql.DB) (bool, error) {
	var value int32
	flyApp := GetFlyAppName(userId, project, task)
	err := db.QueryRow("SELECT CASE WHEN EXISTS (SELECT * FROM FlyMachine WHERE flyApp = ?) THEN 1 ELSE 0 END AS X;", flyApp).Scan(&value)
	if err != nil {
		return false, err
	}
	if value == 0 {
		return false, nil
	} else {
		return true, nil
	}
}