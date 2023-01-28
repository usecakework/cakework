package main

import (
	"database/sql"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/usecakework/cakework/lib/fly"
	"github.com/usecakework/cakework/lib/types"
)

// Get logs for a request
// just returns the whole json blob from logtail as a string for now
// TODO clean up input parameters
func getRequestLogs(userId string, appName string, taskName string, machineId string, requestId string) (*types.RequestLogs, error) {

	flyAppName := fly.GetFlyAppName(userId, appName, taskName)

	searchString := "fulltext:" + machineId + " fulltext:" + flyAppName + " fly.app.instance=" + machineId

	logs, err := getLogs(searchString)

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// Get details about a request. Returns nil if request is not found.
func getRun(db *sql.DB, requestId string) (*types.Request, error) {
	// TODO use the userId and app
	var request types.Request
	var result sql.NullString
	var machineId sql.NullString
	var createdAt time.Time
	var updatedAt time.Time
	err := db.QueryRow("SELECT userId, app, task, parameters, requestId, machineId, status, result, createdAt, updatedAt FROM TaskRun where requestId = ?", requestId).Scan(&request.UserId, &request.App, &request.Task, &request.Parameters, &request.RequestId, &machineId, &request.Status, &result, &createdAt, &updatedAt)
	if err != nil {
		log.Debug("Got an error trying to query")
		log.Error(err)
		if err.Error() == sql.ErrNoRows.Error() {
			log.Debug("go no rows")
			
		}
		return nil, err
	}

	if result.Valid {
		request.Result = result.String
	}

	if machineId.Valid {
		request.MachineId = machineId.String
	}

	request.CreatedAt = createdAt.Unix()
	request.UpdatedAt = updatedAt.Unix()

	return &request, nil
}
