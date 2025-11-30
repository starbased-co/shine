package main

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/creachadair/jrpc2/handler"
	"github.com/starbased-co/shine/pkg/rpc"
)

// TestNotificationHandlers_PrismStarted tests prism started notification handler
func TestNotificationHandlers_PrismStarted(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.PrismStartedNotification

	mux := handler.Map{
		"notify/prism/started": handler.New(func(ctx context.Context, n *rpc.PrismStartedNotification) (*NotificationAck, error) {
			receivedNotification = n
			return &NotificationAck{}, nil
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

	notification := &rpc.PrismStartedNotification{
		Panel: "panel-0",
		Name:  "clock",
		PID:   1234,
	}

	var ack NotificationAck
	if err := client.Call(context.Background(), "notify/prism/started", notification, &ack); err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.Panel != "panel-0" {
		t.Errorf("panel = %q, want panel-0", receivedNotification.Panel)
	}
}

// TestNotificationHandlers_PrismStopped tests prism stopped notification handler
func TestNotificationHandlers_PrismStopped(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.PrismStoppedNotification

	mux := handler.Map{
		"notify/prism/stopped": handler.New(func(ctx context.Context, n *rpc.PrismStoppedNotification) (*NotificationAck, error) {
			receivedNotification = n
			return &NotificationAck{}, nil
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

	notification := &rpc.PrismStoppedNotification{
		Panel:    "panel-0",
		Name:     "clock",
		ExitCode: 0,
	}

	var ack NotificationAck
	if err := client.Call(context.Background(), "notify/prism/stopped", notification, &ack); err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", receivedNotification.ExitCode)
	}
}

// TestNotificationHandlers_PrismCrashed tests prism crashed notification handler
func TestNotificationHandlers_PrismCrashed(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.PrismCrashedNotification

	mux := handler.Map{
		"notify/prism/crashed": handler.New(func(ctx context.Context, n *rpc.PrismCrashedNotification) (*NotificationAck, error) {
			receivedNotification = n
			return &NotificationAck{}, nil
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

	notification := &rpc.PrismCrashedNotification{
		Panel:    "panel-0",
		Name:     "clock",
		ExitCode: 1,
		Signal:   9,
	}

	var ack NotificationAck
	if err := client.Call(context.Background(), "notify/prism/crashed", notification, &ack); err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.Signal != 9 {
		t.Errorf("Signal = %d, want 9", receivedNotification.Signal)
	}
}

// TestNotificationHandlers_SurfaceSwitched tests surface switched notification handler
func TestNotificationHandlers_SurfaceSwitched(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var receivedNotification *rpc.SurfaceSwitchedNotification

	mux := handler.Map{
		"notify/surface/switched": handler.New(func(ctx context.Context, n *rpc.SurfaceSwitchedNotification) (*NotificationAck, error) {
			receivedNotification = n
			return &NotificationAck{}, nil
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

	notification := &rpc.SurfaceSwitchedNotification{
		Panel: "panel-0",
		From:  "clock",
		To:    "bar",
	}

	var ack NotificationAck
	if err := client.Call(context.Background(), "notify/surface/switched", notification, &ack); err != nil {
		t.Fatalf("notification call error: %v", err)
	}

	if receivedNotification == nil {
		t.Fatal("notification not received")
	}
	if receivedNotification.From != "clock" {
		t.Errorf("From = %q, want clock", receivedNotification.From)
	}
}

// TestNotificationHandlers_Integration tests full notification flow
func TestNotificationHandlers_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	receivedCount := 0
	mux := handler.Map{
		"notify/prism/started": handler.New(func(ctx context.Context, n *rpc.PrismStartedNotification) (*NotificationAck, error) {
			receivedCount++
			return &NotificationAck{}, nil
		}),
		"notify/prism/stopped": handler.New(func(ctx context.Context, n *rpc.PrismStoppedNotification) (*NotificationAck, error) {
			receivedCount++
			return &NotificationAck{}, nil
		}),
		"notify/prism/crashed": handler.New(func(ctx context.Context, n *rpc.PrismCrashedNotification) (*NotificationAck, error) {
			receivedCount++
			return &NotificationAck{}, nil
		}),
		"notify/surface/switched": handler.New(func(ctx context.Context, n *rpc.SurfaceSwitchedNotification) (*NotificationAck, error) {
			receivedCount++
			return &NotificationAck{}, nil
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
	var ack NotificationAck

	// Send all notification types
	client.Call(ctx, "notify/prism/started", &rpc.PrismStartedNotification{Panel: "panel-0", Name: "clock", PID: 1}, &ack)
	client.Call(ctx, "notify/prism/stopped", &rpc.PrismStoppedNotification{Panel: "panel-0", Name: "clock"}, &ack)
	client.Call(ctx, "notify/prism/crashed", &rpc.PrismCrashedNotification{Panel: "panel-0", Name: "bar"}, &ack)
	client.Call(ctx, "notify/surface/switched", &rpc.SurfaceSwitchedNotification{Panel: "panel-0", From: "clock", To: "bar"}, &ack)

	if receivedCount != 4 {
		t.Errorf("received %d notifications, want 4", receivedCount)
	}
}

// TestNotificationHandlers_ConcurrentNotifications tests concurrent notification delivery
func TestNotificationHandlers_ConcurrentNotifications(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	var notificationCount int32
	mux := handler.Map{
		"notify/prism/started": handler.New(func(ctx context.Context, n *rpc.PrismStartedNotification) (*NotificationAck, error) {
			atomic.AddInt32(&notificationCount, 1)
			return &NotificationAck{}, nil
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

	// Send concurrent notifications
	done := make(chan error, len(clients))
	for i, c := range clients {
		go func(idx int, client *rpc.ShinedClient) {
			var ack NotificationAck
			err := client.Call(ctx, "notify/prism/started", &rpc.PrismStartedNotification{
				Panel: "panel-0",
				Name:  "test",
				PID:   1000 + idx,
			}, &ack)
			done <- err
		}(i, c)
	}

	// Wait for all notifications
	for i := 0; i < len(clients); i++ {
		if err := <-done; err != nil {
			t.Errorf("client %d notification error: %v", i, err)
		}
	}

	time.Sleep(50 * time.Millisecond)

	count := atomic.LoadInt32(&notificationCount)
	if count != int32(len(clients)) {
		t.Errorf("received %d notifications, want %d", count, len(clients))
	}
}

// TestNotificationHandlers_ErrorHandling tests error handling in notification handlers
func TestNotificationHandlers_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "shine.sock")

	// Handler that returns error
	mux := handler.Map{
		"notify/prism/started": handler.New(func(ctx context.Context, n *rpc.PrismStartedNotification) (*NotificationAck, error) {
			if n.Name == "invalid" {
				return nil, rpc.ErrInvalidParams("invalid prism name")
			}
			return &NotificationAck{}, nil
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

	// Valid notification should succeed
	var ack NotificationAck
	err = client.Call(ctx, "notify/prism/started", &rpc.PrismStartedNotification{
		Panel: "panel-0",
		Name:  "clock",
		PID:   1234,
	}, &ack)
	if err != nil {
		t.Errorf("valid notification error: %v", err)
	}

	// Invalid notification should fail
	err = client.Call(ctx, "notify/prism/started", &rpc.PrismStartedNotification{
		Panel: "panel-0",
		Name:  "invalid",
		PID:   1234,
	}, &ack)
	if err == nil {
		t.Error("invalid notification should return error")
	}
}
