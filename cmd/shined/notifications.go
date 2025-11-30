package main

import (
	"context"
	"log"

	"github.com/starbased-co/shine/pkg/rpc"
)

type NotificationAck struct{}

func (h *Handlers) handlePrismStarted(ctx context.Context, n *rpc.PrismStartedNotification) (*NotificationAck, error) {
	log.Printf("[%s] prism started: %s (PID %d)", n.Panel, n.Name, n.PID)

	if h.state != nil {
		h.state.OnPanelPrismStarted(n.Panel, n.Name, n.PID)
	}

	return &NotificationAck{}, nil
}

func (h *Handlers) handlePrismStopped(ctx context.Context, n *rpc.PrismStoppedNotification) (*NotificationAck, error) {
	log.Printf("[%s] prism stopped: %s (exit=%d)", n.Panel, n.Name, n.ExitCode)

	if h.state != nil {
		h.state.OnPanelPrismStopped(n.Panel, n.Name, n.ExitCode)
	}

	h.pm.MarkPrismStopped(n.Panel, n.Name, n.ExitCode)

	return &NotificationAck{}, nil
}

func (h *Handlers) handlePrismCrashed(ctx context.Context, n *rpc.PrismCrashedNotification) (*NotificationAck, error) {
	log.Printf("[%s] prism CRASHED: %s (exit=%d, signal=%d)", n.Panel, n.Name, n.ExitCode, n.Signal)

	if h.state != nil {
		h.state.OnPanelPrismCrashed(n.Panel, n.Name, n.ExitCode, n.Signal)
	}

	// Trigger restart policy based on exit code
	h.pm.TriggerRestartPolicy(n.Panel, n.Name, n.ExitCode)

	return &NotificationAck{}, nil
}

// handleSurfaceSwitched handles notification when foreground prism changes
func (h *Handlers) handleSurfaceSwitched(ctx context.Context, n *rpc.SurfaceSwitchedNotification) (*NotificationAck, error) {
	log.Printf("[%s] surface switched: %s → %s", n.Panel, n.From, n.To)

	// Update state tracking if needed
	if h.state != nil {
		h.state.OnPanelSurfaceSwitched(n.Panel, n.From, n.To)
	}

	return &NotificationAck{}, nil
}
