package main

type Request struct {
	RequestId  string `json:"request"`
	UserId     string `json:"userId"`
	App        string `json:"app"`
	Task       string `json:"task"`
	Status     string `json:"status"`
	Parameters string `json:"parameters"`
	Result     string `json:"result"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type RequestLogLine struct {
	Timestamp string `json:"dt"`
	LogLevel  string `json:"log.level"`
	Message   string `json:"message"`
}

type RequestLogs struct {
	LogLines []RequestLogLine `json:"data"`
}
