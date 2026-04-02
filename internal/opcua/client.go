// Package opcua manages OPC-UA client connections and subscriptions.
package opcua

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/monitor"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
)

// DataChange represents a value change notification from an OPC-UA node.
type DataChange struct {
	NodeID    string
	Value     interface{}
	Timestamp time.Time
	Status    ua.StatusCode
}

// NodeConfig describes a single OPC-UA node to subscribe to.
type NodeConfig struct {
	NodeID   string
	Interval time.Duration
}

// ClientConfig holds configuration for an OPC-UA client connection.
type ClientConfig struct {
	Endpoint     string
	SecurityMode ua.MessageSecurityMode
	Nodes        []NodeConfig
}

// Client manages a single OPC-UA connection and its subscriptions.
type Client struct {
	config   ClientConfig
	logger   *zap.Logger
	onChange func(DataChange)

	mu       sync.Mutex
	client   *opcua.Client
	monitor  *monitor.NodeMonitor
	sub      *monitor.Subscription
	cancelFn context.CancelFunc
	closed   bool
}

// NewClient creates a new OPC-UA client manager.
func NewClient(config ClientConfig, logger *zap.Logger, onChange func(DataChange)) *Client {
	return &Client{
		config:   config,
		logger:   logger,
		onChange: onChange,
	}
}

// Connect establishes a connection to the OPC-UA server and starts subscriptions.
// It blocks until the context is cancelled or an unrecoverable error occurs.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	return c.connectWithRetry(ctx)
}

func (c *Client) connectWithRetry(ctx context.Context) error {
	attempt := 0
	for {
		err := c.tryConnect(ctx)
		if err == nil {
			attempt = 0
			err = c.subscribe(ctx)
			if err == nil {
				return nil
			}
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		attempt++
		backoff := calculateBackoff(attempt)
		c.logger.Warn("connection failed, retrying",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoff),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
}

func (c *Client) tryConnect(ctx context.Context) error {
	opts := []opcua.Option{
		opcua.SecurityMode(c.config.SecurityMode),
	}

	client, err := opcua.NewClient(c.config.Endpoint, opts...)
	if err != nil {
		return fmt.Errorf("creating OPC-UA client: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connecting to %s: %w", c.config.Endpoint, err)
	}

	c.mu.Lock()
	c.client = client
	c.mu.Unlock()

	c.logger.Info("connected to OPC-UA server", zap.String("endpoint", c.config.Endpoint))
	return nil
}

func (c *Client) subscribe(ctx context.Context) error {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		return fmt.Errorf("not connected")
	}

	m, err := monitor.NewNodeMonitor(client)
	if err != nil {
		return fmt.Errorf("creating node monitor: %w", err)
	}

	c.mu.Lock()
	c.monitor = m
	c.mu.Unlock()

	subCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancelFn = cancel
	c.mu.Unlock()

	nodeIDs := make([]string, len(c.config.Nodes))
	for i, n := range c.config.Nodes {
		nodeIDs[i] = n.NodeID
	}

	ch := make(chan *monitor.DataChangeMessage, 256)

	sub, err := m.ChanSubscribe(
		subCtx,
		&opcua.SubscriptionParameters{Interval: defaultInterval(c.config.Nodes)},
		ch,
		nodeIDs...,
	)
	if err != nil {
		cancel()
		return fmt.Errorf("subscribing to nodes: %w", err)
	}

	c.mu.Lock()
	c.sub = sub
	c.mu.Unlock()

	c.logger.Info("subscribed to nodes",
		zap.Int("count", len(nodeIDs)),
		zap.String("endpoint", c.config.Endpoint),
	)

	go c.processDataChanges(subCtx, ch)

	<-subCtx.Done()
	return subCtx.Err()
}

func (c *Client) processDataChanges(ctx context.Context, ch <-chan *monitor.DataChangeMessage) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.Error != nil {
				c.logger.Warn("data change error",
					zap.String("nodeID", msg.NodeID.String()),
					zap.Error(msg.Error),
				)
				continue
			}

			dc := DataChange{
				NodeID:    msg.NodeID.String(),
				Value:     msg.Value.Value(),
				Timestamp: msg.SourceTimestamp,
				Status:    msg.Status,
			}
			c.onChange(dc)
		}
	}
}

// Close shuts down the client, closing all subscriptions and the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true

	if c.cancelFn != nil {
		c.cancelFn()
	}

	if c.sub != nil {
		c.sub.Unsubscribe(context.Background())
	}

	if c.client != nil {
		if err := c.client.Close(context.Background()); err != nil {
			c.logger.Warn("error closing OPC-UA client", zap.Error(err))
			return err
		}
	}

	c.logger.Info("OPC-UA client closed", zap.String("endpoint", c.config.Endpoint))
	return nil
}

// calculateBackoff returns the wait duration with exponential backoff and jitter.
// Initial: 1s, max: 60s.
func calculateBackoff(attempt int) time.Duration {
	base := math.Pow(2, float64(attempt-1))
	if base > 60 {
		base = 60
	}
	jitter := rand.Float64() * base * 0.5
	return time.Duration((base+jitter)*1000) * time.Millisecond
}

func defaultInterval(nodes []NodeConfig) time.Duration {
	if len(nodes) == 0 {
		return 5 * time.Second
	}
	min := nodes[0].Interval
	for _, n := range nodes[1:] {
		if n.Interval < min {
			min = n.Interval
		}
	}
	if min <= 0 {
		return 5 * time.Second
	}
	return min
}
