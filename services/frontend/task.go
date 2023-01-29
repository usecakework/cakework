package main

import (
	"database/sql"
	"time"

	"github.com/usecakework/cakework/lib/types"
)

func GetTaskLogs(db *sql.DB, userId string, project string, taskName string, statusFilter string) (types.TaskLogs, error) {
	var runs []types.Run

	var rows *sql.Rows
	var err error

	if statusFilter == "" {
		rows, err = db.Query("SELECT runId, status, parameters, result, createdAt, updatedAt FROM TaskRun where userId = ? AND project = ? AND task = ? ORDER BY updatedAt DESC LIMIT 100", userId, project, taskName)
	} else {
		rows, err = db.Query("SELECT runId, status, parameters, result, createdAt, updatedAt FROM TaskRun where userId = ? AND project = ? AND task = ? AND status = ? ORDER BY updatedAt DESC LIMIT 100", userId, project, taskName, statusFilter)
	}

	if err != nil {
		return types.TaskLogs{
			Runs: runs,
		}, err
	}

	defer rows.Close()

	for rows.Next() {
		var result sql.NullString
		var createdAt time.Time
		var updatedAt time.Time
		var run types.Run
		if err := rows.Scan(&run.RunId, &run.Status, &run.Parameters, &result, &createdAt, &updatedAt); err != nil {
			return types.TaskLogs{Runs: runs}, err
		}
		if result.Valid {
			run.Result = result.String
		}
		run.CreatedAt = createdAt.Unix()
		run.UpdatedAt = updatedAt.Unix()
		runs = append(runs, run)
	}

	if err = rows.Err(); err != nil {
		return types.TaskLogs{
			Runs: runs,
		}, err
	}

	return types.TaskLogs{
		Runs: runs,
	}, nil
}
