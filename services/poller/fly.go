package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/usecakework/cakework/lib/fly/api"
)

func GetLatestImage(flyApp string, db *sql.DB) (string, error) {
	var config api.MachineConfig
	err := db.QueryRow("SELECT image, machineId FROM FlyMachine WHERE flyApp = ? ORDER BY createdAt DESC LIMIT 1", flyApp).Scan(&config.Config.Image, &config.MachineId)
	if err != nil {
		return "", err
	}

	fmt.Println(config) // TODO delete

	if config.Config.Image != "" {
		return config.Config.Image, nil
	} else {
		return "", errors.New("Got the latest deployed FlyMachine, but image is null")
	}
}