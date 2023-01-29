package main

import (
	"database/sql"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/usecakework/cakework/lib/fly"
	"github.com/usecakework/cakework/lib/types"
)

// Get logs for a run
// just returns the whole json blob from logtail as a string for now
// TODO clean up input parameters
func getRunLogs(userId string, project string, taskName string, machineId string, runId string) (*types.RunLogs, error) {

	flyAppName := fly.GetFlyAppName(userId, project, taskName)

	searchString := "fulltext:" + machineId + " fulltext:" + flyAppName + " fly.app.instance=" + machineId

	logs, err := getLogs(searchString)

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// Get details about a request. Returns nil if request is not found.
func getRun(db *sql.DB, runId string) (*types.Run, error) {
	// TODO use the userId and app
	var run types.Run
	var result sql.NullString
	var machineId sql.NullString
	var createdAt time.Time
	var updatedAt time.Time
	err := db.QueryRow("SELECT userId, project, task, parameters, runId, machineId, status, result, createdAt, updatedAt FROM Run where runId = ?", runId).Scan(&run.UserId, &run.Project, &run.Task, &run.Parameters, &run.RunId, &machineId, &run.Status, &result, &createdAt, &updatedAt)
	if err != nil {
		log.Error("Got an error trying to query")
		log.Error(err)
		if err.Error() == sql.ErrNoRows.Error() {
			log.Debug("go no rows")
			return nil, nil
		}
		return nil, err
	}

	if result.Valid {
		run.Result = result.String
	}

	if machineId.Valid {
		run.MachineId = machineId.String
	}

	run.CreatedAt = createdAt.Unix()
	run.UpdatedAt = updatedAt.Unix()

	return &run, nil
}
