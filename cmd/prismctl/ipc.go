package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/starbased-co/shine/pkg/paths"
	"github.com/starbased-co/shine/pkg/rpc"
)

func startRPCServer(instance string, supervisor *supervisor, stateMgr *StateManager) (*rpc.Server, error) {
	// ensure directory 
	runtimeDir := paths.RuntimeDir()
	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create runtime directory: %w", err)
	}

	socketPath := paths.PrismSocket(instance)

	handlers := newRPCHandlers(supervisor, stateMgr)

	opts := &jrpc2.ServerOptions{
		Logger: func(text string) {
			log.Printf("RPC: %s", text)
		},
	}

	server := rpc.NewServer(socketPath, handlers, opts)

	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("failed to start RPC server: %w", err)
	}

	log.Printf("RPC server listening on: %s", socketPath)

	return server, nil
}

// stopRPCServer gracefully stops the RPC server
func stopRPCServer(server *rpc.Server) {
	if server == nil {
		return
	}

	log.Printf("Stopping RPC server...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Warning: error stopping RPC server: %v", err)
	}

	log.Printf("RPC server stopped")
}
