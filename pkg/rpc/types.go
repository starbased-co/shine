package rpc

// PrismInfo describes a prism's current state
type PrismInfo struct {
	Name     string `json:"name"`
	PID      int    `json:"pid"`
	State    string `json:"state"`     // "fg" or "bg"
	UptimeMs int64  `json:"uptime_ms"` // milliseconds since start
	Restarts int    `json:"restarts"`  // restart count
}

// PanelInfo describes a panel's current state
type PanelInfo struct {
	Instance string `json:"instance"` // unique panel identifier
	Name     string `json:"name"`     // human-readable name
	PID      int    `json:"pid"`      // prismctl process PID
	Socket   string `json:"socket"`   // path to prismctl socket
	Healthy  bool   `json:"healthy"`  // health check status
}

// prismctl RPC types (prism-{instance}.sock)

// UpRequest is the request for prism/up
type UpRequest struct {
	Name string `json:"name"`
}

// UpResult is the response for prism/up
type UpResult struct {
	PID   int    `json:"pid"`
	State string `json:"state"` // "fg" or "bg"
}

// DownRequest is the request for prism/down
type DownRequest struct {
	Name string `json:"name"`
}

// DownResult is the response for prism/down
type DownResult struct {
	Stopped bool `json:"stopped"`
}

// FgRequest is the request for prism/fg
type FgRequest struct {
	Name string `json:"name"`
}

// FgResult is the response for prism/fg
type FgResult struct {
	OK    bool `json:"ok"`
	WasFg bool `json:"was_fg"` // true if already foreground (idempotent)
}

// BgRequest is the request for prism/bg
type BgRequest struct {
	Name string `json:"name"`
}

// BgResult is the response for prism/bg
type BgResult struct {
	OK    bool `json:"ok"`
	WasBg bool `json:"was_bg"` // true if already background (idempotent)
}

// AppInfo describes an app configuration for the configure RPC
type AppInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`    // resolved binary path
	Enabled bool   `json:"enabled"`
}

// ConfigureRequest is the request for prism/configure
type ConfigureRequest struct {
	Apps []AppInfo `json:"apps"`
}

// ConfigureResult is the response for prism/configure
type ConfigureResult struct {
	Started []string `json:"started"` // apps that were started
	Failed  []string `json:"failed"`  // apps that failed to start
}

// ListResult is the response for prism/list
type ListResult struct {
	Prisms []PrismInfo `json:"prisms"`
}

// HealthResult is the response for service/health
type HealthResult struct {
	Healthy    bool `json:"healthy"`
	PrismCount int  `json:"prism_count"`
}

// ShutdownRequest is the request for service/shutdown
type ShutdownRequest struct {
	Graceful bool `json:"graceful"`
}

// ShutdownResult is the response for service/shutdown
type ShutdownResult struct {
	ShuttingDown bool `json:"shutting_down"`
}

// shined RPC types (shine.sock)

// PanelListResult is the response for panel/list
type PanelListResult struct {
	Panels []PanelInfo `json:"panels"`
}

// PanelSpawnRequest is the request for panel/spawn
type PanelSpawnRequest struct {
	Config map[string]any `json:"config"` // panel configuration
}

// PanelSpawnResult is the response for panel/spawn
type PanelSpawnResult struct {
	Instance string `json:"instance"`
	Socket   string `json:"socket"`
}

// PanelKillRequest is the request for panel/kill
type PanelKillRequest struct {
	Instance string `json:"instance"`
}

// PanelKillResult is the response for panel/kill
type PanelKillResult struct {
	Killed bool `json:"killed"`
}

// ServiceStatusResult is the response for service/status
type ServiceStatusResult struct {
	Panels  []PanelInfo `json:"panels"`
	Uptime  int64       `json:"uptime_ms"`
	Version string      `json:"version"`
}

// ConfigReloadResult is the response for config/reload
type ConfigReloadResult struct {
	Reloaded bool     `json:"reloaded"`
	Errors   []string `json:"errors,omitempty"`
}

// Notification types (prismctl → shined)

// PrismStartedNotification is sent when a prism starts
type PrismStartedNotification struct {
	Panel string `json:"panel"` // panel instance
	Name  string `json:"name"`  // prism name
	PID   int    `json:"pid"`
}

// PrismStoppedNotification is sent when a prism stops normally
type PrismStoppedNotification struct {
	Panel    string `json:"panel"`
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
}

// PrismCrashedNotification is sent when a prism crashes unexpectedly
type PrismCrashedNotification struct {
	Panel    string `json:"panel"`
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
	Signal   int    `json:"signal,omitempty"`
}

// SurfaceSwitchedNotification is sent when foreground prism changes
type SurfaceSwitchedNotification struct {
	Panel string `json:"panel"`
	From  string `json:"from"` // previous foreground prism
	To    string `json:"to"`   // new foreground prism
}
