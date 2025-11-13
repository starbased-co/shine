package ipc

// Command represents a command sent via IPC
type Command struct {
	Action string `json:"action"` // "start", "kill", "status", "stop"
	Prism  string `json:"prism,omitempty"`
}

// Response represents a response from IPC
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// StatusResponse represents the status command response
type StatusResponse struct {
	Foreground string        `json:"foreground"`
	Background []string      `json:"background"`
	Prisms     []PrismStatus `json:"prisms"`
}

// PrismStatus represents individual prism status
type PrismStatus struct {
	Name  string `json:"name"`
	PID   int    `json:"pid"`
	State string `json:"state"` // "foreground" or "background"
}
