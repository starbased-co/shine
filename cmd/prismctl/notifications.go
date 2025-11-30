package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/starbased-co/shine/pkg/paths"
	"github.com/starbased-co/shine/pkg/rpc"
)

// with graceful degradation when shined is unavailable
type NotificationManager struct {
	mu         sync.Mutex
	client     *rpc.ShinedClient
	instance   string
	connected  bool
	reconnectC chan struct{}
	stopC      chan struct{}
}

func newNotificationManager(instance string) *NotificationManager {
	nm := &NotificationManager{
		instance:   instance,
		connected:  false,
		reconnectC: make(chan struct{}, 1),
		stopC:      make(chan struct{}),
	}

	// Start background connection loop
	go nm.connectionLoop()

	return nm
}

// connectionLoop maintains connection+reconnection to shined
func (nm *NotificationManager) connectionLoop() {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-nm.stopC:
			return
		case <-nm.reconnectC:
			// Connection attempt triggered
		case <-time.After(backoff):
			// Periodic retry if disconnected
		}

		nm.mu.Lock()
		connected := nm.connected
		nm.mu.Unlock()

		if connected {
			backoff = time.Second
			time.Sleep(5 * time.Second)
			continue
		}

		// Attempt connection
		sockPath := paths.ShinedSocket()
		client, err := rpc.NewShinedClient(sockPath, rpc.WithTimeout(2*time.Second))

		nm.mu.Lock()
		if err != nil {
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			log.Printf("Notification: failed to connect to shined: %v (retry in %v)", err, backoff)
			nm.client = nil
			nm.connected = false
		} else {
			// Connection successful
			log.Printf("Notification: connected to shined at %s", sockPath)
			nm.client = client
			nm.connected = true
			backoff = time.Second
		}
		nm.mu.Unlock()
	}
}

// tryReconnect signals the connection loop to attempt reconnection
func (nm *NotificationManager) tryReconnect() {
	select {
	case nm.reconnectC <- struct{}{}:
	default:
	}
}

func (nm *NotificationManager) sendNotification(fn func(context.Context, *rpc.ShinedClient) error) {
	nm.mu.Lock()
	client := nm.client
	connected := nm.connected
	nm.mu.Unlock()

	if !connected || client == nil {
		// Not connected - silently skip
		return
	}

	// Sends notification (no expectation of response, 0.5s timeout is generous for catching immediate error)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := fn(ctx, client); err != nil {
		log.Printf("Notification: failed to send: %v", err)

		// Mark as disconnected and trigger reconnect
		nm.mu.Lock()
		if nm.client != nil {
			nm.client.Close()
			nm.client = nil
		}
		nm.connected = false
		nm.mu.Unlock()

		nm.tryReconnect()
	}
}

// OnPrismStarted notifies shined that a prism started
func (nm *NotificationManager) OnPrismStarted(name string, pid int) {
	log.Printf("Notification: prism started %s (PID %d)", name, pid)
	nm.sendNotification(func(ctx context.Context, c *rpc.ShinedClient) error {
		return c.NotifyPrismStarted(ctx, nm.instance, name, pid)
	})
}

// OnPrismStopped notifies shined that a prism stopped normally
func (nm *NotificationManager) OnPrismStopped(name string, exitCode int) {
	log.Printf("Notification: prism stopped %s (exit=%d)", name, exitCode)
	nm.sendNotification(func(ctx context.Context, c *rpc.ShinedClient) error {
		return c.NotifyPrismStopped(ctx, nm.instance, name, exitCode)
	})
}

// OnPrismCrashed notifies shined that a prism crashed
func (nm *NotificationManager) OnPrismCrashed(name string, exitCode, signal int) {
	log.Printf("Notification: prism crashed %s (exit=%d, signal=%d)", name, exitCode, signal)
	nm.sendNotification(func(ctx context.Context, c *rpc.ShinedClient) error {
		return c.NotifyPrismCrashed(ctx, nm.instance, name, exitCode, signal)
	})
}

func (nm *NotificationManager) OnForegroundChanged(from, to string) {
	log.Printf("Notification: foreground changed %s → %s", from, to)
	nm.sendNotification(func(ctx context.Context, c *rpc.ShinedClient) error {
		return c.NotifyForegroundChanged(ctx, nm.instance, from, to)
	})
}

func (nm *NotificationManager) Close() {
	close(nm.stopC)

	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.client != nil {
		nm.client.Close()
		nm.client = nil
	}
	nm.connected = false
}
