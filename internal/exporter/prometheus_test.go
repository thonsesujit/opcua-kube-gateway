package exporter

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestNewPrometheusExporter(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := NewPrometheusExporter(testLogger(), reg)

	if e == nil {
		t.Fatal("NewPrometheusExporter returned nil")
	}

	// Verify health metrics are registered
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	names := make(map[string]bool)
	for _, m := range metrics {
		names[m.GetName()] = true
	}

	expected := []string{
		"opcua_gateway_connections_active",
		"opcua_gateway_subscriptions_active",
		"opcua_gateway_reconnections_total",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected health metric %s to be registered", name)
		}
	}

	// errors_total is a CounterVec — it only appears after at least one label set is used
	e.ErrorsTotal.WithLabelValues("connection")
	metrics, err = reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics after init: %v", err)
	}
	names = make(map[string]bool)
	for _, m := range metrics {
		names[m.GetName()] = true
	}
	if !names["opcua_gateway_errors_total"] {
		t.Error("expected health metric opcua_gateway_errors_total to be registered after label init")
	}
}

func TestRegisterNode(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := NewPrometheusExporter(testLogger(), reg)

	subLabels := SubscriptionLabels{
		Namespace:    "factory-floor",
		Subscription: "pump-monitoring",
		Endpoint:     "opc.tcp://plc:4840",
	}
	nodeLabels := NodeLabels{
		NodeID: "ns=2;s=Temperature",
		Unit:   "celsius",
	}

	err := e.RegisterNode("opcua_", "pump_temperature", subLabels, nodeLabels)
	if err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	// Registering same node again should be a no-op
	err = e.RegisterNode("opcua_", "pump_temperature", subLabels, nodeLabels)
	if err != nil {
		t.Fatalf("duplicate register should not error: %v", err)
	}
}

func TestUpdateNode(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := NewPrometheusExporter(testLogger(), reg)

	subLabels := SubscriptionLabels{
		Namespace:    "default",
		Subscription: "test-sub",
		Endpoint:     "opc.tcp://localhost:4840",
	}
	nodeLabels := NodeLabels{
		NodeID: "ns=2;s=Temperature",
		Unit:   "celsius",
	}

	err := e.RegisterNode("opcua_", "temperature", subLabels, nodeLabels)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	key := NodeMetricKey{
		Namespace:    "default",
		Subscription: "test-sub",
		NodeID:       "ns=2;s=Temperature",
	}

	e.UpdateNode(key, 42.5)

	expected := `
# HELP opcua_temperature OPC-UA node value for temperature
# TYPE opcua_temperature gauge
opcua_temperature{endpoint="opc.tcp://localhost:4840",namespace="default",node_id="ns=2;s=Temperature",subscription="test-sub",unit="celsius"} 42.5
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "opcua_temperature"); err != nil {
		t.Errorf("metric value mismatch: %v", err)
	}
}

func TestCustomPrefix(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := NewPrometheusExporter(testLogger(), reg)

	subLabels := SubscriptionLabels{
		Namespace:    "default",
		Subscription: "test-sub",
		Endpoint:     "opc.tcp://localhost:4840",
	}
	nodeLabels := NodeLabels{
		NodeID: "ns=2;s=Temperature",
		Unit:   "celsius",
	}

	err := e.RegisterNode("factory_", "pump_temp", subLabels, nodeLabels)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	key := NodeMetricKey{
		Namespace:    "default",
		Subscription: "test-sub",
		NodeID:       "ns=2;s=Temperature",
	}
	e.UpdateNode(key, 75.0)

	expected := `
# HELP factory_pump_temp OPC-UA node value for pump_temp
# TYPE factory_pump_temp gauge
factory_pump_temp{endpoint="opc.tcp://localhost:4840",namespace="default",node_id="ns=2;s=Temperature",subscription="test-sub",unit="celsius"} 75
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "factory_pump_temp"); err != nil {
		t.Errorf("custom prefix metric mismatch: %v", err)
	}
}

func TestUnregisterSubscription(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := NewPrometheusExporter(testLogger(), reg)

	subLabels := SubscriptionLabels{
		Namespace:    "default",
		Subscription: "test-sub",
		Endpoint:     "opc.tcp://localhost:4840",
	}

	err := e.RegisterNode("opcua_", "temp", subLabels, NodeLabels{NodeID: "ns=2;s=Temp", Unit: "C"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	err = e.RegisterNode("opcua_", "pressure", subLabels, NodeLabels{NodeID: "ns=2;s=Press", Unit: "bar"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	e.UnregisterSubscription("default", "test-sub")

	// Metrics should be removed from internal tracking
	e.mu.RLock()
	count := len(e.metrics)
	e.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 metrics after unregister, got %d", count)
	}
}

func TestUpdateNonExistentNode(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := NewPrometheusExporter(testLogger(), reg)

	key := NodeMetricKey{
		Namespace:    "default",
		Subscription: "missing",
		NodeID:       "ns=2;s=NonExistent",
	}

	// Should not panic, just log a warning
	e.UpdateNode(key, 99.0)
}

func TestHealthMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := NewPrometheusExporter(testLogger(), reg)

	e.ConnectionsActive.Set(3)
	e.SubscriptionsActive.Set(5)
	e.ReconnectionsTotal.Inc()
	e.ReconnectionsTotal.Inc()
	e.ErrorsTotal.WithLabelValues("connection").Inc()

	val := testutil.ToFloat64(e.ConnectionsActive)
	if val != 3 {
		t.Errorf("expected connections_active=3, got %v", val)
	}

	val = testutil.ToFloat64(e.SubscriptionsActive)
	if val != 5 {
		t.Errorf("expected subscriptions_active=5, got %v", val)
	}

	val = testutil.ToFloat64(e.ReconnectionsTotal)
	if val != 2 {
		t.Errorf("expected reconnections_total=2, got %v", val)
	}

	val = testutil.ToFloat64(e.ErrorsTotal.WithLabelValues("connection"))
	if val != 1 {
		t.Errorf("expected errors_total{type=connection}=1, got %v", val)
	}
}
