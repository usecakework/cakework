package main

import (
	"database/sql"
)

type Request struct {
	RequestId  string `json:"request"`
	Status     string `json:"status"`
	Parameters string `json:"parameters"`
	Result     string `json:"result"`
}

type TaskStatus struct {
	Requests []Request `json:"requests"`
}

func getTaskStatus(db *sql.DB, userId string, appName string, taskName string) (TaskStatus, error) {
	var requests []Request

	rows, err := db.Query("SELECT requestId, status, parameters, result FROM TaskRun where userId = ? AND app = ? AND task = ? LIMIT 100", userId, appName, taskName)
	if err != nil {
		return TaskStatus{
			Requests: requests,
		}, err
	}

	defer rows.Close()

	for rows.Next() {
		var result sql.NullString
		var request Request
		if err := rows.Scan(&request.RequestId, &request.Status, &request.Parameters, &result); err != nil {
			return TaskStatus{Requests: requests}, err
		}
		if result.Valid {
			request.Result = result.String
		}
		requests = append(requests, request)
	}

	if err = rows.Err(); err != nil {
		return TaskStatus{
			Requests: requests,
		}, err
	}

	return TaskStatus{
		Requests: requests,
	}, nil
}
