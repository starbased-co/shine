package rpc

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/creachadair/jrpc2/handler"
)

func TestServerClientRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	// Create handlers - jrpc2 requires struct params, not primitives
	type EchoRequest struct {
		Msg string `json:"msg"`
	}
	type AddRequest struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	mux := handler.Map{
		"echo": handler.New(func(ctx context.Context, req *EchoRequest) (string, error) {
			return req.Msg, nil
		}),
		"add": handler.New(func(ctx context.Context, req *AddRequest) (int, error) {
			return req.A + req.B, nil
		}),
	}

	// Start server
	srv := NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	// Wait for server to be ready
	time.Sleep(10 * time.Millisecond)

	// Create client
	client, err := NewClient(sockPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test echo
	var echoResult string
	if err := client.Call(ctx, "echo", map[string]string{"msg": "hello"}, &echoResult); err != nil {
		t.Fatalf("echo call error: %v", err)
	}
	if echoResult != "hello" {
		t.Errorf("echo result = %q, want %q", echoResult, "hello")
	}

	// Test add
	var addResult int
	if err := client.Call(ctx, "add", map[string]int{"a": 3, "b": 5}, &addResult); err != nil {
		t.Fatalf("add call error: %v", err)
	}
	if addResult != 8 {
		t.Errorf("add result = %d, want 8", addResult)
	}
}

func TestServerMethodNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	mux := handler.Map{}
	srv := NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	client, err := NewClient(sockPath)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	defer client.Close()

	var result string
	err = client.Call(context.Background(), "nonexistent", nil, &result)
	if err == nil {
		t.Fatal("expected error for nonexistent method")
	}
}

func TestMultipleClients(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	counter := 0
	mux := handler.Map{
		"increment": handler.New(func(ctx context.Context) (int, error) {
			counter++
			return counter, nil
		}),
	}

	srv := NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	// Create multiple clients
	clients := make([]*Client, 3)
	for i := range clients {
		c, err := NewClient(sockPath)
		if err != nil {
			t.Fatalf("NewClient() error: %v", err)
		}
		clients[i] = c
		defer c.Close()
	}

	// Each client makes a call
	for i, c := range clients {
		var result int
		if err := c.Call(context.Background(), "increment", nil, &result); err != nil {
			t.Fatalf("client %d call error: %v", i, err)
		}
		if result != i+1 {
			t.Errorf("client %d result = %d, want %d", i, result, i+1)
		}
	}
}

func TestServerStop(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	mux := handler.Map{}
	srv := NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if !srv.Running() {
		t.Error("Running() = false, want true")
	}

	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	if srv.Running() {
		t.Error("Running() = true after stop, want false")
	}

	// Second stop should be no-op
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop() error: %v", err)
	}
}

func TestPrismClientMethods(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	// Mock handlers
	mux := handler.Map{
		"prism/up": handler.New(func(ctx context.Context, req *UpRequest) (*UpResult, error) {
			return &UpResult{PID: 1234, State: "fg"}, nil
		}),
		"prism/down": handler.New(func(ctx context.Context, req *DownRequest) (*DownResult, error) {
			return &DownResult{Stopped: true}, nil
		}),
		"prism/fg": handler.New(func(ctx context.Context, req *FgRequest) (*FgResult, error) {
			return &FgResult{OK: true, WasFg: false}, nil
		}),
		"prism/bg": handler.New(func(ctx context.Context, req *BgRequest) (*BgResult, error) {
			return &BgResult{OK: true, WasBg: true}, nil
		}),
		"prism/list": handler.New(func(ctx context.Context) (*ListResult, error) {
			return &ListResult{
				Prisms: []PrismInfo{
					{Name: "clock", PID: 1001, State: "fg"},
					{Name: "bar", PID: 1002, State: "bg"},
				},
			}, nil
		}),
		"service/health": handler.New(func(ctx context.Context) (*HealthResult, error) {
			return &HealthResult{Healthy: true, PrismCount: 2}, nil
		}),
		"service/shutdown": handler.New(func(ctx context.Context, req *ShutdownRequest) (*ShutdownResult, error) {
			return &ShutdownResult{ShuttingDown: true}, nil
		}),
	}

	srv := NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	client, err := NewPrismClient(sockPath)
	if err != nil {
		t.Fatalf("NewPrismClient() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test Up
	upResult, err := client.Up(ctx, "clock")
	if err != nil {
		t.Fatalf("Up() error: %v", err)
	}
	if upResult.PID != 1234 {
		t.Errorf("Up().PID = %d, want 1234", upResult.PID)
	}

	// Test Down
	downResult, err := client.Down(ctx, "clock")
	if err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	if !downResult.Stopped {
		t.Error("Down().Stopped = false, want true")
	}

	// Test Fg
	fgResult, err := client.Fg(ctx, "bar")
	if err != nil {
		t.Fatalf("Fg() error: %v", err)
	}
	if !fgResult.OK {
		t.Error("Fg().OK = false, want true")
	}

	// Test Bg
	bgResult, err := client.Bg(ctx, "clock")
	if err != nil {
		t.Fatalf("Bg() error: %v", err)
	}
	if !bgResult.WasBg {
		t.Error("Bg().WasBg = false, want true")
	}

	// Test List
	listResult, err := client.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(listResult.Prisms) != 2 {
		t.Errorf("List().Prisms len = %d, want 2", len(listResult.Prisms))
	}

	// Test Health
	healthResult, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if !healthResult.Healthy {
		t.Error("Health().Healthy = false, want true")
	}

	// Test Shutdown
	shutdownResult, err := client.Shutdown(ctx, true)
	if err != nil {
		t.Fatalf("Shutdown() error: %v", err)
	}
	if !shutdownResult.ShuttingDown {
		t.Error("Shutdown().ShuttingDown = false, want true")
	}
}

func TestShinedClientMethods(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	mux := handler.Map{
		"panel/list": handler.New(func(ctx context.Context) (*PanelListResult, error) {
			return &PanelListResult{
				Panels: []PanelInfo{
					{Instance: "panel-0", Name: "main", PID: 2001, Healthy: true},
				},
			}, nil
		}),
		"panel/spawn": handler.New(func(ctx context.Context, req *PanelSpawnRequest) (*PanelSpawnResult, error) {
			return &PanelSpawnResult{
				Instance: "panel-1",
				Socket:   "/run/user/1000/shine/prism-panel-1.sock",
			}, nil
		}),
		"panel/kill": handler.New(func(ctx context.Context, req *PanelKillRequest) (*PanelKillResult, error) {
			return &PanelKillResult{Killed: true}, nil
		}),
		"service/status": handler.New(func(ctx context.Context) (*ServiceStatusResult, error) {
			return &ServiceStatusResult{
				Panels:  []PanelInfo{{Instance: "panel-0", Healthy: true}},
				Uptime:  60000,
				Version: "0.2.0",
			}, nil
		}),
		"config/reload": handler.New(func(ctx context.Context) (*ConfigReloadResult, error) {
			return &ConfigReloadResult{Reloaded: true}, nil
		}),
	}

	srv := NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	client, err := NewShinedClient(sockPath)
	if err != nil {
		t.Fatalf("NewShinedClient() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test ListPanels
	listResult, err := client.ListPanels(ctx)
	if err != nil {
		t.Fatalf("ListPanels() error: %v", err)
	}
	if len(listResult.Panels) != 1 {
		t.Errorf("ListPanels().Panels len = %d, want 1", len(listResult.Panels))
	}

	// Test SpawnPanel
	spawnResult, err := client.SpawnPanel(ctx, map[string]any{"name": "test"})
	if err != nil {
		t.Fatalf("SpawnPanel() error: %v", err)
	}
	if spawnResult.Instance != "panel-1" {
		t.Errorf("SpawnPanel().Instance = %q, want %q", spawnResult.Instance, "panel-1")
	}

	// Test KillPanel
	killResult, err := client.KillPanel(ctx, "panel-0")
	if err != nil {
		t.Fatalf("KillPanel() error: %v", err)
	}
	if !killResult.Killed {
		t.Error("KillPanel().Killed = false, want true")
	}

	// Test Status
	statusResult, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if statusResult.Version != "0.2.0" {
		t.Errorf("Status().Version = %q, want %q", statusResult.Version, "0.2.0")
	}

	// Test Reload
	reloadResult, err := client.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload() error: %v", err)
	}
	if !reloadResult.Reloaded {
		t.Error("Reload().Reloaded = false, want true")
	}
}

func TestClientTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "nonexistent.sock")

	// Try to connect to non-existent socket
	_, err := NewClient(sockPath, WithTimeout(100*time.Millisecond))
	if err == nil {
		t.Fatal("expected error connecting to non-existent socket")
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"PrismNotFound", ErrPrismNotFound("test"), CodePrismNotFound},
		{"PrismNotRunning", ErrPrismNotRunning("test"), CodePrismNotRunning},
		{"PrismAlreadyUp", ErrPrismAlreadyUp("test"), CodePrismAlreadyUp},
		{"PanelNotFound", ErrPanelNotFound("test"), CodePanelNotFound},
		{"ShuttingDown", ErrShuttingDown(), CodeShuttingDown},
		{"Config", ErrConfig("bad config"), CodeConfigError},
		{"ResourceBusy", ErrResourceBusy("socket"), CodeResourceBusy},
		{"InvalidParams", ErrInvalidParams("missing name"), CodeInvalidParams},
		{"Internal", ErrInternal(nil), CodeInternal},
		{"NotImplemented", ErrNotImplemented("method"), CodeNotImplemented},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("error is nil")
			}
		})
	}
}
