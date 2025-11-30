package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/creachadair/jrpc2/handler"
	"github.com/starbased-co/shine/pkg/rpc"
)

// TestNotificationManager_Reconnect tests manual reconnection trigger
func TestNotificationManager_Reconnect(t *testing.T) {
	nm := &NotificationManager{
		instance:   "panel-test",
		connected:  false,
		reconnectC: make(chan struct{}, 1),
		stopC:      make(chan struct{}),
	}
	defer close(nm.stopC)

	// Trigger reconnect
	nm.tryReconnect()

	// Should not block (channel is buffered)
	select {
	case <-nm.reconnectC:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("tryReconnect() did not signal reconnectC")
	}

	// Multiple calls should not block
	nm.tryReconnect()
	nm.tryReconnect()
}

// TestNotificationDelivery_PrismStarted tests prism started notification
func TestNotificationDelivery_PrismStarted(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.PrismStartedNotification
	mux := handler.Map{
		"notify/prism/started": handler.New(func(ctx context.Context, n *rpc.PrismStartedNotification) (*struct{}, error) {
			receivedNotification = n
			return &struct{}{}, nil
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

	// Send notification
	notification := &rpc.PrismStartedNotification{
		Panel: "panel-0",
		Name:  "clock",
		PID:   1234,
	}

	var ack struct{}
	err = client.Call(context.Background(), "notify/prism/started", notification, &ack)
	if err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.Panel != "panel-0" {
		t.Errorf("panel = %q, want panel-0", receivedNotification.Panel)
	}
	if receivedNotification.Name != "clock" {
		t.Errorf("name = %q, want clock", receivedNotification.Name)
	}
	if receivedNotification.PID != 1234 {
		t.Errorf("PID = %d, want 1234", receivedNotification.PID)
	}
}

// TestNotificationDelivery_PrismStopped tests prism stopped notification
func TestNotificationDelivery_PrismStopped(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.PrismStoppedNotification
	mux := handler.Map{
		"notify/prism/stopped": handler.New(func(ctx context.Context, n *rpc.PrismStoppedNotification) (*struct{}, error) {
			receivedNotification = n
			return &struct{}{}, nil
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

	// Send notification
	notification := &rpc.PrismStoppedNotification{
		Panel:    "panel-0",
		Name:     "clock",
		ExitCode: 0,
	}

	var ack struct{}
	err = client.Call(context.Background(), "notify/prism/stopped", notification, &ack)
	if err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", receivedNotification.ExitCode)
	}
}

// TestNotificationDelivery_PrismCrashed tests prism crashed notification
func TestNotificationDelivery_PrismCrashed(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.PrismCrashedNotification
	mux := handler.Map{
		"notify/prism/crashed": handler.New(func(ctx context.Context, n *rpc.PrismCrashedNotification) (*struct{}, error) {
			receivedNotification = n
			return &struct{}{}, nil
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

	// Send notification
	notification := &rpc.PrismCrashedNotification{
		Panel:    "panel-0",
		Name:     "clock",
		ExitCode: 1,
		Signal:   9,
	}

	var ack struct{}
	err = client.Call(context.Background(), "notify/prism/crashed", notification, &ack)
	if err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", receivedNotification.ExitCode)
	}
	if receivedNotification.Signal != 9 {
		t.Errorf("Signal = %d, want 9", receivedNotification.Signal)
	}
}

// TestNotificationDelivery_SurfaceSwitched tests surface switched notification
func TestNotificationDelivery_SurfaceSwitched(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.SurfaceSwitchedNotification
	mux := handler.Map{
		"notify/surface/switched": handler.New(func(ctx context.Context, n *rpc.SurfaceSwitchedNotification) (*struct{}, error) {
			receivedNotification = n
			return &struct{}{}, nil
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

	// Send notification
	notification := &rpc.SurfaceSwitchedNotification{
		Panel: "panel-0",
		From:  "clock",
		To:    "bar",
	}

	var ack struct{}
	err = client.Call(context.Background(), "notify/surface/switched", notification, &ack)
	if err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.From != "clock" {
		t.Errorf("From = %q, want clock", receivedNotification.From)
	}
	if receivedNotification.To != "bar" {
		t.Errorf("To = %q, want bar", receivedNotification.To)
	}
}

// TestNotificationBatch tests multiple notifications in sequence
func TestNotificationBatch(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	notifications := make([]string, 0)
	mux := handler.Map{
		"notify/prism/started": handler.New(func(ctx context.Context, n *rpc.PrismStartedNotification) (*struct{}, error) {
			notifications = append(notifications, "started:"+n.Name)
			return &struct{}{}, nil
		}),
		"notify/prism/stopped": handler.New(func(ctx context.Context, n *rpc.PrismStoppedNotification) (*struct{}, error) {
			notifications = append(notifications, "stopped:"+n.Name)
			return &struct{}{}, nil
		}),
		"notify/surface/switched": handler.New(func(ctx context.Context, n *rpc.SurfaceSwitchedNotification) (*struct{}, error) {
			notifications = append(notifications, "switched:"+n.From+"->"+n.To)
			return &struct{}{}, nil
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

	// Send sequence of notifications
	var ack struct{}

	client.Call(ctx, "notify/prism/started", &rpc.PrismStartedNotification{
		Panel: "panel-0", Name: "clock", PID: 1,
	}, &ack)

	client.Call(ctx, "notify/prism/started", &rpc.PrismStartedNotification{
		Panel: "panel-0", Name: "bar", PID: 2,
	}, &ack)

	client.Call(ctx, "notify/surface/switched", &rpc.SurfaceSwitchedNotification{
		Panel: "panel-0", From: "clock", To: "bar",
	}, &ack)

	client.Call(ctx, "notify/prism/stopped", &rpc.PrismStoppedNotification{
		Panel: "panel-0", Name: "clock", ExitCode: 0,
	}, &ack)

	// Verify all notifications received in order
	expected := []string{
		"started:clock",
		"started:bar",
		"switched:clock->bar",
		"stopped:clock",
	}

	if len(notifications) != len(expected) {
		t.Errorf("received %d notifications, want %d", len(notifications), len(expected))
	}

	for i, want := range expected {
		if i < len(notifications) && notifications[i] != want {
			t.Errorf("notification[%d] = %q, want %q", i, notifications[i], want)
		}
	}
}
