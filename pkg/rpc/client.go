package rpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
)

type Client struct {
	sockPath string
	conn     net.Conn
	client   *jrpc2.Client
	timeout  time.Duration
}

type ClientOption func(*Client)

func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = d
	}
}

func NewClient(sockPath string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		sockPath: sockPath,
		timeout:  5 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	conn, err := net.DialTimeout("unix", sockPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", sockPath, err)
	}

	ch := channel.Line(conn, conn)
	c.conn = conn
	c.client = jrpc2.NewClient(ch, nil)

	return c, nil
}

func (c *Client) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	return c.client.CallResult(ctx, method, params, result)
}

func (c *Client) Notify(ctx context.Context, method string, params any) error {
	return c.client.Notify(ctx, method, params)
}

type PrismClient struct {
	*Client
}

func NewPrismClient(sockPath string, opts ...ClientOption) (*PrismClient, error) {
	c, err := NewClient(sockPath, opts...)
	if err != nil {
		return nil, err
	}
	return &PrismClient{Client: c}, nil
}

func (c *PrismClient) Up(ctx context.Context, name string) (*UpResult, error) {
	var result UpResult
	err := c.Call(ctx, "prism/up", &UpRequest{Name: name}, &result)
	return &result, err
}

func (c *PrismClient) Down(ctx context.Context, name string) (*DownResult, error) {
	var result DownResult
	err := c.Call(ctx, "prism/down", &DownRequest{Name: name}, &result)
	return &result, err
}

func (c *PrismClient) Fg(ctx context.Context, name string) (*FgResult, error) {
	var result FgResult
	err := c.Call(ctx, "prism/fg", &FgRequest{Name: name}, &result)
	return &result, err
}

func (c *PrismClient) Bg(ctx context.Context, name string) (*BgResult, error) {
	var result BgResult
	err := c.Call(ctx, "prism/bg", &BgRequest{Name: name}, &result)
	return &result, err
}

func (c *PrismClient) List(ctx context.Context) (*ListResult, error) {
	var result ListResult
	err := c.Call(ctx, "prism/list", nil, &result)
	return &result, err
}

func (c *PrismClient) Health(ctx context.Context) (*HealthResult, error) {
	var result HealthResult
	err := c.Call(ctx, "service/health", nil, &result)
	return &result, err
}

func (c *PrismClient) Shutdown(ctx context.Context, graceful bool) (*ShutdownResult, error) {
	var result ShutdownResult
	err := c.Call(ctx, "service/shutdown", &ShutdownRequest{Graceful: graceful}, &result)
	return &result, err
}

func (c *PrismClient) Configure(ctx context.Context, apps []AppInfo) (*ConfigureResult, error) {
	var result ConfigureResult
	err := c.Call(ctx, "prism/configure", &ConfigureRequest{Apps: apps}, &result)
	return &result, err
}

type ShinedClient struct {
	*Client
}

func NewShinedClient(sockPath string, opts ...ClientOption) (*ShinedClient, error) {
	c, err := NewClient(sockPath, opts...)
	if err != nil {
		return nil, err
	}
	return &ShinedClient{Client: c}, nil
}

func (c *ShinedClient) ListPanels(ctx context.Context) (*PanelListResult, error) {
	var result PanelListResult
	err := c.Call(ctx, "panel/list", nil, &result)
	return &result, err
}

func (c *ShinedClient) SpawnPanel(ctx context.Context, config map[string]any) (*PanelSpawnResult, error) {
	var result PanelSpawnResult
	err := c.Call(ctx, "panel/spawn", &PanelSpawnRequest{Config: config}, &result)
	return &result, err
}

func (c *ShinedClient) KillPanel(ctx context.Context, instance string) (*PanelKillResult, error) {
	var result PanelKillResult
	err := c.Call(ctx, "panel/kill", &PanelKillRequest{Instance: instance}, &result)
	return &result, err
}

func (c *ShinedClient) Status(ctx context.Context) (*ServiceStatusResult, error) {
	var result ServiceStatusResult
	err := c.Call(ctx, "service/status", nil, &result)
	return &result, err
}

func (c *ShinedClient) Reload(ctx context.Context) (*ConfigReloadResult, error) {
	var result ConfigReloadResult
	err := c.Call(ctx, "config/reload", nil, &result)
	return &result, err
}

func (c *ShinedClient) NotifyPrismStarted(ctx context.Context, panel, name string, pid int) error {
	return c.Notify(ctx, "prism/started", &PrismStartedNotification{
		Panel: panel,
		Name:  name,
		PID:   pid,
	})
}

func (c *ShinedClient) NotifyPrismStopped(ctx context.Context, panel, name string, exitCode int) error {
	return c.Notify(ctx, "prism/stopped", &PrismStoppedNotification{
		Panel:    panel,
		Name:     name,
		ExitCode: exitCode,
	})
}

func (c *ShinedClient) NotifyPrismCrashed(ctx context.Context, panel, name string, exitCode, signal int) error {
	return c.Notify(ctx, "prism/crashed", &PrismCrashedNotification{
		Panel:    panel,
		Name:     name,
		ExitCode: exitCode,
		Signal:   signal,
	})
}

func (c *ShinedClient) NotifyForegroundChanged(ctx context.Context, panel, from, to string) error {
	return c.Notify(ctx, "foreground/changed", &ForegroundChangedNotification{
		Panel: panel,
		From:  from,
		To:    to,
	})
}
