package main

import (
	"context"
	"log"

	"github.com/creachadair/jrpc2/handler"
	"github.com/starbased-co/shine/pkg/rpc"
)

// rpcHandlers holds references to supervisor and state for handling RPC calls
type rpcHandlers struct {
	supervisor   *supervisor
	stateManager *StateManager
}

// newRPCHandlers creates RPC method handlers
func newRPCHandlers(sup *supervisor, stateMgr *StateManager) handler.Map {
	h := &rpcHandlers{
		supervisor:   sup,
		stateManager: stateMgr,
	}

	return handler.Map{
		"prism/configure":  handler.New(h.handleConfigure),
		"prism/up":         handler.New(h.handleUp),
		"prism/down":       handler.New(h.handleDown),
		"prism/fg":         handler.New(h.handleFg),
		"prism/bg":         handler.New(h.handleBg),
		"prism/list":       handler.New(h.handleList),
		"service/health":   handler.New(h.handleHealth),
		"service/shutdown": handler.New(h.handleShutdown),
	}
}

// handleConfigure receives app configuration from shined and starts apps
func (h *rpcHandlers) handleConfigure(ctx context.Context, req *rpc.ConfigureRequest) (*rpc.ConfigureResult, error) {
	log.Printf("RPC: prism/configure with %d apps", len(req.Apps))

	result := &rpc.ConfigureResult{
		Started: make([]string, 0),
		Failed:  make([]string, 0),
	}

	for _, app := range req.Apps {
		if !app.Enabled {
			continue
		}

		// Register the resolved path for this app
		h.supervisor.registerApp(app.Name, app.Path)

		// Start the app (first one becomes foreground, rest background)
		if err := h.supervisor.start(app.Name); err != nil {
			log.Printf("Failed to start app %s: %v", app.Name, err)
			result.Failed = append(result.Failed, app.Name)
			continue
		}

		result.Started = append(result.Started, app.Name)
	}

	return result, nil
}

// handleUp starts or resumes a prism (idempotent)
func (h *rpcHandlers) handleUp(ctx context.Context, req *rpc.UpRequest) (*rpc.UpResult, error) {
	if req.Name == "" {
		return nil, rpc.ErrInvalidParams("name is required")
	}

	log.Printf("RPC: prism/up %s", req.Name)

	if err := h.supervisor.start(req.Name); err != nil {
		return nil, rpc.ErrOperationFailed("start", err)
	}

	// Get current state after start
	h.supervisor.mu.Lock()
	idx := h.supervisor.findPrism(req.Name)
	if idx == -1 {
		h.supervisor.mu.Unlock()
		return nil, rpc.ErrPrismNotFound(req.Name)
	}

	prism := h.supervisor.prismList[idx]
	state := "bg"
	if idx == 0 {
		state = "fg"
	}
	h.supervisor.mu.Unlock()

	return &rpc.UpResult{
		PID:   prism.pid,
		State: state,
	}, nil
}

// handleDown stops a prism
func (h *rpcHandlers) handleDown(ctx context.Context, req *rpc.DownRequest) (*rpc.DownResult, error) {
	if req.Name == "" {
		return nil, rpc.ErrInvalidParams("name is required")
	}

	log.Printf("RPC: prism/down %s", req.Name)

	if err := h.supervisor.killPrism(req.Name); err != nil {
		return nil, rpc.ErrOperationFailed("kill", err)
	}

	return &rpc.DownResult{
		Stopped: true,
	}, nil
}

// handleFg brings prism to foreground (idempotent)
func (h *rpcHandlers) handleFg(ctx context.Context, req *rpc.FgRequest) (*rpc.FgResult, error) {
	if req.Name == "" {
		return nil, rpc.ErrInvalidParams("name is required")
	}

	log.Printf("RPC: prism/fg %s", req.Name)

	h.supervisor.mu.Lock()
	idx := h.supervisor.findPrism(req.Name)

	if idx == -1 {
		h.supervisor.mu.Unlock()
		return nil, rpc.ErrPrismNotFound(req.Name)
	}

	// Already foreground?
	if idx == 0 {
		h.supervisor.mu.Unlock()
		return &rpc.FgResult{
			OK:    true,
			WasFg: true,
		}, nil
	}

	// Resume to foreground
	h.supervisor.mu.Unlock()
	if err := h.supervisor.start(req.Name); err != nil {
		return nil, rpc.ErrOperationFailed("foreground", err)
	}

	return &rpc.FgResult{
		OK:    true,
		WasFg: false,
	}, nil
}

// handleBg sends prism to background (idempotent, not implemented yet)
func (h *rpcHandlers) handleBg(ctx context.Context, req *rpc.BgRequest) (*rpc.BgResult, error) {
	if req.Name == "" {
		return nil, rpc.ErrInvalidParams("name is required")
	}

	log.Printf("RPC: prism/bg %s (not implemented - all non-foreground prisms are background)", req.Name)

	h.supervisor.mu.Lock()
	defer h.supervisor.mu.Unlock()

	idx := h.supervisor.findPrism(req.Name)
	if idx == -1 {
		return nil, rpc.ErrPrismNotFound(req.Name)
	}

	// If not foreground, already in background
	if idx != 0 {
		return &rpc.BgResult{
			OK:    true,
			WasBg: true,
		}, nil
	}

	// Cannot send foreground to background without bringing another forward
	// This would require "suspend foreground" semantics which we don't have yet
	return &rpc.BgResult{
		OK:    false,
		WasBg: false,
	}, nil
}

// handleList returns list of all prisms with their states
func (h *rpcHandlers) handleList(ctx context.Context) (*rpc.ListResult, error) {
	log.Printf("RPC: prism/list")

	h.supervisor.mu.Lock()
	defer h.supervisor.mu.Unlock()

	prisms := make([]rpc.PrismInfo, 0, len(h.supervisor.prismList))
	for i, p := range h.supervisor.prismList {
		state := "bg"
		if i == 0 {
			state = "fg"
		}

		prisms = append(prisms, rpc.PrismInfo{
			Name:     p.name,
			PID:      p.pid,
			State:    state,
			UptimeMs: 0, // TODO: track start time
			Restarts: 0, // TODO: track restarts
		})
	}

	return &rpc.ListResult{
		Prisms: prisms,
	}, nil
}

// handleHealth returns health check status
func (h *rpcHandlers) handleHealth(ctx context.Context) (*rpc.HealthResult, error) {
	log.Printf("RPC: service/health")

	h.supervisor.mu.Lock()
	defer h.supervisor.mu.Unlock()

	return &rpc.HealthResult{
		Healthy:    !h.supervisor.shuttingDown,
		PrismCount: len(h.supervisor.prismList),
	}, nil
}

// handleShutdown initiates graceful shutdown
func (h *rpcHandlers) handleShutdown(ctx context.Context, req *rpc.ShutdownRequest) (*rpc.ShutdownResult, error) {
	log.Printf("RPC: service/shutdown (graceful=%v)", req.Graceful)

	// Trigger shutdown in background
	go h.supervisor.shutdown()

	return &rpc.ShutdownResult{
		ShuttingDown: true,
	}, nil
}
