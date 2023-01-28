package types

// TODO: organize so that we put structs in their relevant files

type Request struct {
	RequestId  string `json:"requestId"`
	UserId     string `json:"userId"`
	App        string `json:"app"`
	Task       string `json:"task"`
	Status     string `json:"status"`
	Parameters string `json:"parameters"`
	Result     string `json:"result"`
	CPU        int    `json:"cpu"`
	MemoryMB   int    `json:"memory"`
	MachineId  string `json:"machineId"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type RunRequest struct {
	Task       string `json:"task"`
	Parameters []interface{} `json:"parameters"`
	CPU        int    `json:"cpu"`
	Memory   int    `json:"memory"`
}

type Task struct {
	RequestId  string `json:"requestId"`
	UserId     string `json:"userId"`
	App        string `json:"app"`
	Task       string `json:"task"`
	Status     string `json:"status"`
	Parameters string `json:"parameters"`
	Result     string `json:"result"`
	CPU        int    `json:"cpu"`
	MemoryMB   int    `json:"memory"`
	MachineId  string `json:"machineId"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type Compute struct {
	CPU      string `json:"cpu"`
	MemoryMB string `json:"memoryMB"`
}

// TODO Timestamp is a string since that's what logtail gives us. Should force to int64 on the server instead of making clients deal with it.
type RequestLogLine struct {
	Timestamp string `json:"_dt"`
	LogLevel  string `json:"log.level"`
	Message   string `json:"message"`
}

type RequestLogs struct {
	LogLines []RequestLogLine `json:"data"`
}

type CreateTokenRequest struct {
	UserId string `json:"userId"`
	Name   string `json:"name"`
}

type CreateUserRequest struct {
	UserId string `json:"userId"`
}

type GetUserRequest struct {
	UserId string `json:"userId"`
}

type User struct {
	Id string `json:"id"`
}

type ClientToken struct {
	Token string `json:"token"`
}

type GetStatusRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
}

type GetStatusResponse struct {
	Status string `json:"status"`
}

type GetTaskLogsRequest struct {
	UserId       string `json:"userId"`
	App          string `json:"app"`
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

type GetResultRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
}

// Q: how will errors be handled? TODO need to expose an error field?
type GetResultResponse struct {
	Result string `json:"result"`
}

// this currently updates the TaskRun table
type UpdateMachineId struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
	MachineId string `json:"machineId"`
}

type UpdateStatusRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
	Status    string `json:"status"`
}

type UpdateResultRequest struct {
	UserId    string `json:"userId"`
	App       string `json:"app"`
	RequestId string `json:"requestId"`
	Result    string `json:"result"`
}

type CreateClientTokenRequest struct {
	UserId string `json:"userId"` // Q: can we get the user id from the auth info?
	Name   string `json:"name"`
}

type GetUserByClientTokenRequest struct {
	Token string `json:"token"`
}

type GetRequestLogsRequest struct {
	UserId    string `json:"userId"`
	RequestId string `json:"requestId"`
}

type TaskLogs struct {
	Requests []Request `json:"requests"`
}

type CLISecrets struct {
	FLY_ACCESS_TOKEN string `json:"FLY_ACCESS_TOKEN"`
}
