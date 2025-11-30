package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/creachadair/jrpc2/handler"
	"github.com/starbased-co/shine/pkg/rpc"
)

// TestShinedIPC_PanelLifecycle tests full panel spawn → list → kill flow
func TestShinedIPC_PanelLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	// Create test handlers with mock implementations
	listResult := &rpc.PanelListResult{Panels: []rpc.PanelInfo{}}
	spawnedPanel := &rpc.PanelSpawnResult{}
	panelExists := false

	mux := handler.Map{
		"panel/list": handler.New(func(ctx context.Context) (*rpc.PanelListResult, error) {
			return listResult, nil
		}),
		"panel/spawn": handler.New(func(ctx context.Context, req *rpc.PanelSpawnRequest) (*rpc.PanelSpawnResult, error) {
			if panelExists {
				return nil, rpc.ErrResourceBusy("panel already exists")
			}
			panelExists = true
			spawnedPanel = &rpc.PanelSpawnResult{
				Instance: "panel-0",
				Socket:   "/tmp/test.sock",
			}
			listResult.Panels = append(listResult.Panels, rpc.PanelInfo{
				Instance: "panel-0",
				Name:     "test-panel",
				PID:      1234,
				Healthy:  true,
			})
			return spawnedPanel, nil
		}),
		"panel/kill": handler.New(func(ctx context.Context, req *rpc.PanelKillRequest) (*rpc.PanelKillResult, error) {
			if !panelExists {
				return nil, rpc.ErrPanelNotFound(req.Instance)
			}
			panelExists = false
			listResult.Panels = []rpc.PanelInfo{}
			return &rpc.PanelKillResult{Killed: true}, nil
		}),
	}

	// Start server
	srv := rpc.NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	// Create client
	client, err := rpc.NewShinedClient(sockPath)
	if err != nil {
		t.Fatalf("NewShinedClient() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test 1: List panels (should be empty)
	list, err := client.ListPanels(ctx)
	if err != nil {
		t.Fatalf("ListPanels() error: %v", err)
	}
	if len(list.Panels) != 0 {
		t.Errorf("initial ListPanels() = %d panels, want 0", len(list.Panels))
	}

	// Test 2: Spawn panel
	spawn, err := client.SpawnPanel(ctx, map[string]any{
		"name":     "test-panel",
		"instance": "panel-0",
		"enabled":  true,
	})
	if err != nil {
		t.Fatalf("SpawnPanel() error: %v", err)
	}
	if spawn.Instance == "" {
		t.Error("SpawnPanel() returned empty instance")
	}

	// Test 3: List panels (should have one)
	list, err = client.ListPanels(ctx)
	if err != nil {
		t.Fatalf("ListPanels() after spawn error: %v", err)
	}
	if len(list.Panels) != 1 {
		t.Errorf("ListPanels() after spawn = %d panels, want 1", len(list.Panels))
	}

	// Test 4: Kill panel
	kill, err := client.KillPanel(ctx, spawn.Instance)
	if err != nil {
		t.Fatalf("KillPanel() error: %v", err)
	}
	if !kill.Killed {
		t.Error("KillPanel() returned Killed=false")
	}

	// Test 5: List panels (should be empty again)
	list, err = client.ListPanels(ctx)
	if err != nil {
		t.Fatalf("ListPanels() after kill error: %v", err)
	}
	if len(list.Panels) != 0 {
		t.Errorf("ListPanels() after kill = %d panels, want 0", len(list.Panels))
	}
}

// TestShinedIPC_ServiceStatus tests service status endpoint
func TestShinedIPC_ServiceStatus(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	mux := handler.Map{
		"service/status": handler.New(func(ctx context.Context) (*rpc.ServiceStatusResult, error) {
			return &rpc.ServiceStatusResult{
				Panels: []rpc.PanelInfo{
					{Instance: "panel-0", Healthy: true},
				},
				Uptime:  60000,
				Version: "0.2.0",
			}, nil
		}),
	}

	srv := rpc.NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	client, err := rpc.NewShinedClient(sockPath)
	if err != nil {
		t.Fatalf("NewShinedClient() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get status
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}

	if status.Version == "" {
		t.Error("Status().Version is empty")
	}
	if status.Uptime == 0 {
		t.Error("Status().Uptime is 0")
	}
}

// TestShinedIPC_ConfigReload tests config reload endpoint
func TestShinedIPC_ConfigReload(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	reloaded := false
	mux := handler.Map{
		"config/reload": handler.New(func(ctx context.Context) (*rpc.ConfigReloadResult, error) {
			reloaded = true
			return &rpc.ConfigReloadResult{Reloaded: true}, nil
		}),
	}

	srv := rpc.NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	client, err := rpc.NewShinedClient(sockPath)
	if err != nil {
		t.Fatalf("NewShinedClient() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	result, err := client.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload() error: %v", err)
	}

	if !result.Reloaded {
		t.Error("Reload() returned Reloaded=false")
	}
	if !reloaded {
		t.Error("reload handler was not called")
	}
}

// TestShinedIPC_ErrorHandling tests error handling for invalid requests
func TestShinedIPC_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	panelExists := false

	mux := handler.Map{
		"panel/spawn": handler.New(func(ctx context.Context, req *rpc.PanelSpawnRequest) (*rpc.PanelSpawnResult, error) {
			if req.Config["name"] == nil {
				return nil, rpc.ErrInvalidParams("name is required")
			}
			if panelExists {
				return nil, rpc.ErrResourceBusy("panel already exists")
			}
			panelExists = true
			return &rpc.PanelSpawnResult{Instance: "panel-0"}, nil
		}),
		"panel/kill": handler.New(func(ctx context.Context, req *rpc.PanelKillRequest) (*rpc.PanelKillResult, error) {
			if !panelExists {
				return nil, rpc.ErrPanelNotFound(req.Instance)
			}
			return &rpc.PanelKillResult{Killed: true}, nil
		}),
	}

	srv := rpc.NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	client, err := rpc.NewShinedClient(sockPath)
	if err != nil {
		t.Fatalf("NewShinedClient() error: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		testFn  func() error
		wantErr bool
	}{
		{
			name: "spawn with empty config",
			testFn: func() error {
				_, err := client.SpawnPanel(ctx, map[string]any{})
				return err
			},
			wantErr: true,
		},
		{
			name: "kill nonexistent panel",
			testFn: func() error {
				_, err := client.KillPanel(ctx, "nonexistent")
				return err
			},
			wantErr: true,
		},
		{
			name: "spawn duplicate panel",
			testFn: func() error {
				// Spawn first panel
				_, err := client.SpawnPanel(ctx, map[string]any{
					"name":     "test",
					"instance": "panel-0",
					"enabled":  true,
				})
				if err != nil {
					return err
				}

				// Try to spawn duplicate
				_, err = client.SpawnPanel(ctx, map[string]any{
					"name":     "test",
					"instance": "panel-0",
					"enabled":  true,
				})
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFn()
			if (err != nil) != tt.wantErr {
				t.Errorf("test error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestShinedIPC_ConcurrentAccess tests multiple clients accessing the service
func TestShinedIPC_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	callCount := 0
	mux := handler.Map{
		"panel/list": handler.New(func(ctx context.Context) (*rpc.PanelListResult, error) {
			callCount++
			return &rpc.PanelListResult{Panels: []rpc.PanelInfo{}}, nil
		}),
	}

	srv := rpc.NewServer(sockPath, mux, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer srv.Stop(context.Background())

	time.Sleep(10 * time.Millisecond)

	// Create multiple clients
	clients := make([]*rpc.ShinedClient, 5)
	for i := range clients {
		c, err := rpc.NewShinedClient(sockPath)
		if err != nil {
			t.Fatalf("NewShinedClient() %d error: %v", i, err)
		}
		clients[i] = c
		defer c.Close()
	}

	ctx := context.Background()

	// All clients make concurrent requests
	done := make(chan error, len(clients))
	for i, c := range clients {
		go func(idx int, client *rpc.ShinedClient) {
			_, err := client.ListPanels(ctx)
			done <- err
		}(i, c)
	}

	// Wait for all requests to complete
	for i := 0; i < len(clients); i++ {
		if err := <-done; err != nil {
			t.Errorf("client %d error: %v", i, err)
		}
	}

	if callCount != len(clients) {
		t.Errorf("handler called %d times, want %d", callCount, len(clients))
	}
}
