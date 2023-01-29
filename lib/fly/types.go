package fly

type MachineConfig struct {
	Name   string `json:"name,omitempty"`
	Config Config `json:"config,omitempty"`
	MachineId string `json:"id,omitempty"`
}

type Restart struct {
	Policy string `json:"policy,omitempty"`
}

type Config struct {
	Image string `json:"image,omitempty"`
	Guest Guest  `json:"guest,omitempty"`
	Restart Restart `json:"restart,omitempty"`
}

type Guest struct {
	CPUKind  string `json:"cpu_kind,omitempty"`
	CPUs     int    `json:"cpus,omitempty"`
	Memory int    `json:"memory_mb,omitempty"`
}
