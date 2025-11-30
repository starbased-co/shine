package rpc

type PrismInfo struct {
	Name     string `json:"name"`
	PID      int    `json:"pid"`
	State    string `json:"state"`     // "fg" or "bg"
	UptimeMs int64  `json:"uptime_ms"` // milliseconds since start
	Restarts int    `json:"restarts"`  // restart count
}

type PanelInfo struct {
	Instance string `json:"instance"` // unique panel identifier
	Name     string `json:"name"`     // human-readable name
	PID      int    `json:"pid"`      // prismctl process PID
	Socket   string `json:"socket"`   // path to prismctl socket
	Healthy  bool   `json:"healthy"`  // health check status
}

type UpRequest struct {
	Name string `json:"name"`
}

type UpResult struct {
	PID   int    `json:"pid"`
	State string `json:"state"` // "fg" or "bg"
}

type DownRequest struct {
	Name string `json:"name"`
}

type DownResult struct {
	Stopped bool `json:"stopped"`
}

type FgRequest struct {
	Name string `json:"name"`
}

type FgResult struct {
	OK    bool `json:"ok"`
	WasFg bool `json:"was_fg"` // true if already foreground (idempotent)
}

type BgRequest struct {
	Name string `json:"name"`
}

type BgResult struct {
	OK    bool `json:"ok"`
	WasBg bool `json:"was_bg"` // true if already background (idempotent)
}

type AppInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`    // resolved binary path
	Enabled bool   `json:"enabled"`
}

type ConfigureRequest struct {
	Apps []AppInfo `json:"apps"`
}

type ConfigureResult struct {
	Started []string `json:"started"` // apps that were started
	Failed  []string `json:"failed"`  // apps that failed to start
}

type ListResult struct {
	Prisms []PrismInfo `json:"prisms"`
}

type HealthResult struct {
	Healthy    bool `json:"healthy"`
	PrismCount int  `json:"prism_count"`
}

type ShutdownRequest struct {
	Graceful bool `json:"graceful"`
}

type ShutdownResult struct {
	ShuttingDown bool `json:"shutting_down"`
}

type PanelListResult struct {
	Panels []PanelInfo `json:"panels"`
}

type PanelSpawnRequest struct {
	Config map[string]any `json:"config"` // panel configuration
}

type PanelSpawnResult struct {
	Instance string `json:"instance"`
	Socket   string `json:"socket"`
}

type PanelKillRequest struct {
	Instance string `json:"instance"`
}

type PanelKillResult struct {
	Killed bool `json:"killed"`
}

type ServiceStatusResult struct {
	Panels  []PanelInfo `json:"panels"`
	Uptime  int64       `json:"uptime_ms"`
	Version string      `json:"version"`
}

type ConfigReloadResult struct {
	Reloaded bool     `json:"reloaded"`
	Errors   []string `json:"errors,omitempty"`
}

type PrismStartedNotification struct {
	Panel string `json:"panel"` // panel instance
	Name  string `json:"name"`  // prism name
	PID   int    `json:"pid"`
}

type PrismStoppedNotification struct {
	Panel    string `json:"panel"`
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
}

type PrismCrashedNotification struct {
	Panel    string `json:"panel"`
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
	Signal   int    `json:"signal,omitempty"`
}

type ForegroundChangedNotification struct {
	Panel string `json:"panel"`
	From  string `json:"from"` // previous foreground prism
	To    string `json:"to"`   // new foreground prism
}
