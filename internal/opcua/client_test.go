package opcua

import (
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestNewClient(t *testing.T) {
	cfg := ClientConfig{
		Endpoint:     "opc.tcp://localhost:4840",
		SecurityMode: ua.MessageSecurityModeNone,
		Nodes: []NodeConfig{
			{NodeID: "ns=2;s=Temperature", Interval: 1 * time.Second},
		},
	}

	called := false
	onChange := func(dc DataChange) { called = true }

	c := NewClient(cfg, testLogger(), onChange)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.config.Endpoint != cfg.Endpoint {
		t.Errorf("expected endpoint %s, got %s", cfg.Endpoint, c.config.Endpoint)
	}
	if c.closed {
		t.Error("new client should not be closed")
	}

	// Verify onChange is stored (invoke to prove it's wired)
	c.onChange(DataChange{NodeID: "test"})
	if !called {
		t.Error("onChange callback not wired correctly")
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt    int
		minSeconds float64
		maxSeconds float64
	}{
		{1, 0.5, 2.0},   // 2^0 = 1s base, jitter up to 0.5s
		{2, 1.0, 3.5},   // 2^1 = 2s base, jitter up to 1s
		{3, 2.0, 7.0},   // 2^2 = 4s base, jitter up to 2s
		{7, 30.0, 96.0},  // capped at 60s base, jitter up to 30s
		{10, 30.0, 96.0}, // still capped at 60s
	}

	for _, tt := range tests {
		d := calculateBackoff(tt.attempt)
		seconds := d.Seconds()
		if seconds < tt.minSeconds {
			t.Errorf("attempt %d: backoff %v < min %v", tt.attempt, seconds, tt.minSeconds)
		}
		if seconds > tt.maxSeconds {
			t.Errorf("attempt %d: backoff %v > max %v", tt.attempt, seconds, tt.maxSeconds)
		}
	}
}

func TestCalculateBackoff_Jitter(t *testing.T) {
	// Call multiple times and verify we get different values (jitter)
	seen := make(map[time.Duration]bool)
	for i := 0; i < 20; i++ {
		d := calculateBackoff(3)
		seen[d] = true
	}
	if len(seen) < 2 {
		t.Error("expected jitter to produce varying backoff values")
	}
}

func TestDefaultInterval(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []NodeConfig
		expected time.Duration
	}{
		{
			name:     "empty nodes",
			nodes:    nil,
			expected: 5 * time.Second,
		},
		{
			name: "single node",
			nodes: []NodeConfig{
				{NodeID: "ns=2;s=Temp", Interval: 2 * time.Second},
			},
			expected: 2 * time.Second,
		},
		{
			name: "picks minimum",
			nodes: []NodeConfig{
				{NodeID: "ns=2;s=Temp", Interval: 5 * time.Second},
				{NodeID: "ns=2;s=Pressure", Interval: 1 * time.Second},
				{NodeID: "ns=2;s=Status", Interval: 10 * time.Second},
			},
			expected: 1 * time.Second,
		},
		{
			name: "zero interval defaults to 5s",
			nodes: []NodeConfig{
				{NodeID: "ns=2;s=Temp", Interval: 0},
			},
			expected: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultInterval(tt.nodes)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestClient_CloseBeforeConnect(t *testing.T) {
	cfg := ClientConfig{
		Endpoint:     "opc.tcp://localhost:4840",
		SecurityMode: ua.MessageSecurityModeNone,
	}
	c := NewClient(cfg, testLogger(), func(dc DataChange) {})

	err := c.Close()
	if err != nil {
		t.Errorf("close before connect should not error, got: %v", err)
	}
	if !c.closed {
		t.Error("client should be marked as closed")
	}
}

func TestDataChange_Fields(t *testing.T) {
	now := time.Now()
	dc := DataChange{
		NodeID:    "ns=2;s=Temperature",
		Value:     42.5,
		Timestamp: now,
		Status:    ua.StatusOK,
	}

	if dc.NodeID != "ns=2;s=Temperature" {
		t.Errorf("unexpected NodeID: %s", dc.NodeID)
	}
	if dc.Value.(float64) != 42.5 {
		t.Errorf("unexpected Value: %v", dc.Value)
	}
	if dc.Timestamp != now {
		t.Error("timestamp mismatch")
	}
	if dc.Status != ua.StatusOK {
		t.Errorf("unexpected Status: %v", dc.Status)
	}
}
