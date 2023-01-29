package types

type Run struct {
	RunId      string `json:"runId"`
	UserId     string `json:"userId"`
    Project    string `json:"project"`
	Task       string `json:"task"`
	Status     string `json:"status"`
	Parameters string `json:"parameters"`
	Result     string `json:"result"`
	CPU        int    `json:"cpu"`
	Memory     int    `json:"memory"`
	MachineId  string `json:"machineId"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type RunRequest struct {
	Project    string                 `json:"project"`
	Task       string                 `json:"task"`
	Parameters map[string]interface{} `json:"parameters"`
	CPU        int                    `json:"cpu"`
	Memory     int                    `json:"memory"`
}

type Compute struct {
	CPU        string `json:"cpu"`
	Memory     string `json:"memory"`
}

// TODO Timestamp is a string since that's what logtail gives us. Should force to int64 on the server instead of making clients deal with it.
type RunLogLine struct {
	Timestamp  string `json:"_dt"`
	LogLevel   string `json:"log.level"`
	Message    string `json:"message"`
}

type RunLogs struct {
	LogLines []RunLogLine `json:"data"`
}

type CreateTokenRequest struct {
	UserId     string `json:"userId"`
	Name       string `json:"name"`
}

type CreateUserRequest struct {
	UserId     string `json:"userId"`
}

type GetUserRequest struct {
	UserId     string `json:"userId"`
}

type User struct {
	Id         string `json:"id"`
}

type ClientToken struct {
	Token      string `json:"token"`
}

type GetRunStatusRequest struct {
	UserId     string `json:"userId"`
	Project    string `json:"project"`
	RunId      string `json:"runId"`
}

type GetRunStatusResponse struct {
	Status     string `json:"status"`
}

type GetTaskLogsRequest struct {
	UserId       string `json:"userId"`
	Project      string `json:"project"`
	Task         string `json:"task"`
	StatusFilter string `json:"status_filter"`
}

type CreateMachineRequest struct {
	UserId    string `json:"userId"`
	Project   string `json:"project"`
	Task      string `json:"task"`
	FlyApp    string `json:"flyApp"`
	Name      string `json:"name"`
	MachineId string `json:"machineId"`
	State     string `json:"state"`
	Image     string `json:"image"`
	Source    string `json:"source"`
}

// TODO add cpu and memory info and other config info to the machine?
type FlyMachine struct {
	UserId    string `json:"userId"`
	Project   string `json:"project"`
	Task      string `json:"task"`
	FlyApp    string `json:"flyApp"`
	Name      string `json:"name"`
	MachineId string `json:"machineId"`
	State     string `json:"state"`
	Image     string `json:"image"`
	Source    string `json:"source"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type GetRunResultRequest struct {
	UserId        string `json:"userId"`
	Project       string `json:"project"`
	RunId         string `json:"runId"`
}

type GetRunResultResponse struct {
	Result        string `json:"result"`
}

type UpdateMachineIdRequest struct {
	UserId    string `json:"userId"`
	Project   string `json:"project"`
	RunId     string `json:"runId"`
	MachineId string `json:"machineId"`
}

type UpdateRunStatusRequest struct {
	RunId     string `json:"runId"`
	Status    string `json:"status"`
}

type UpdateRunResultRequest struct {
	RunId     string `json:"runId"`
	Result    string `json:"result"`
}

type CreateClientTokenRequest struct {
	UserId string `json:"userId"` // Q: can we get the user id from the auth info?
	Name   string `json:"name"`
}

type GetUserByClientTokenRequest struct {
	Token string `json:"token"`
}

type GetRunLogsRequest struct {
	UserId    string `json:"userId"`
	RunId     string `json:"runId"`
}

type TaskLogs struct {
	Runs []Run `json:"runs"`
}

type CLISecrets struct {
	FLY_ACCESS_TOKEN string `json:"FLY_ACCESS_TOKEN"`
}
