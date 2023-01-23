package main

import (
	"database/sql"
	"time"

	"github.com/usecakework/cakework/lib/fly"
	"github.com/usecakework/cakework/lib/types"
)

// Get logs for a request
// just returns the whole json blob from logtail as a string for now
func getRequestLogs(userId string, appName string, taskName string, requestId string) (*types.RequestLogs, error) {

	// construct search params
	request, err := getRequestDetails(db, requestId)
	if err != nil {
		return nil, err
	}

	flyAppName := fly.GetFlyAppName(userId, appName, taskName)
	searchString := request.MachineId + " " + flyAppName

	logs, err := getLogs(searchString)

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// Get details about a request. Returns nil if request is not found.
func getRequestDetails(db *sql.DB, requestId string) (*types.Request, error) {
	// TODO use the userId and app
	var request types.Request
	var result sql.NullString
	var createdAt time.Time
	var updatedAt time.Time
	err := db.QueryRow("SELECT userId, app, task, parameters, requestId, machineId, status, result, createdAt, updatedAt FROM TaskRun where requestId = ?", requestId).Scan(&request.UserId, &request.App, &request.Task, &request.Parameters, &request.RequestId, &request.MachineId, &request.Status, &result, &createdAt, &updatedAt)
	if err != nil {
		if err.Error() == sql.ErrNoRows.Error() {
			return nil, nil
		} else {
			return nil, err
		}

	}

	if result.Valid {
		request.Result = result.String
	}
	request.CreatedAt = createdAt.Unix()
	request.UpdatedAt = updatedAt.Unix()

	return &request, nil
}
