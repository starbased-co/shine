package main

import (
	"context"
	"log"
	"os"

	"github.com/creachadair/jrpc2/handler"
	"github.com/starbased-co/shine/pkg/paths"
	"github.com/starbased-co/shine/pkg/rpc"
)

var rpcServer *rpc.Server

func startRPCServer(pm *PanelManager, stateMgr *StateManager, cfgPath string) error {
	// ensure directory
	runtimeDir := paths.RuntimeDir()
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}

	h := &Handlers{
		pm:      pm,
		state:   stateMgr,
		cfgPath: cfgPath,
	}

	mux := handler.Map{
		"panel/list":      rpc.HandlerFunc(h.handlePanelList),
		"panel/spawn":     rpc.Handler(h.handlePanelSpawn),
		"panel/kill":      rpc.Handler(h.handlePanelKill),
		"service/status":  rpc.HandlerFunc(h.handleServiceStatus),
		"config/reload":   rpc.HandlerFunc(h.handleConfigReload),
		"prism/started":   rpc.Handler(h.handlePrismStarted),
		"prism/stopped":   rpc.Handler(h.handlePrismStopped),
		"prism/crashed":   rpc.Handler(h.handlePrismCrashed),
		"foreground/changed": rpc.Handler(h.handleForegroundChanged),
	}

	rpcServer = rpc.NewServer(paths.ShinedSocket(), mux, nil)
	if err := rpcServer.Start(); err != nil {
		return err
	}

	log.Printf("RPC server listening on %s", rpcServer.SocketPath())
	return nil
}

func stopRPCServer() {
	if rpcServer != nil {
		log.Println("Stopping RPC server")
		rpcServer.Stop(context.Background())
	}
}
