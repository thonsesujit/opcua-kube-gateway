// Package exporter manages Prometheus metric registration and updates for OPC-UA nodes.
package exporter

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// NodeMetricKey uniquely identifies a metric for a specific OPC-UA node within a subscription.
type NodeMetricKey struct {
	Namespace    string
	Subscription string
	NodeID       string
}

// SubscriptionLabels holds the static labels for a subscription's metrics.
type SubscriptionLabels struct {
	Namespace    string
	Subscription string
	Endpoint     string
}

// NodeLabels holds the labels for a specific node metric.
type NodeLabels struct {
	NodeID string
	Unit   string
}

// PrometheusExporter manages Prometheus metrics for OPC-UA node values.
type PrometheusExporter struct {
	mu       sync.RWMutex
	logger   *zap.Logger
	registry prometheus.Registerer
	gauges   map[string]*prometheus.GaugeVec
	metrics  map[NodeMetricKey]prometheus.Gauge

	// Operator health metrics
	ConnectionsActive   prometheus.Gauge
	SubscriptionsActive prometheus.Gauge
	ReconnectionsTotal  prometheus.Counter
	ErrorsTotal         *prometheus.CounterVec
}

// NewPrometheusExporter creates a new exporter that registers metrics with the given registry.
func NewPrometheusExporter(logger *zap.Logger, registry prometheus.Registerer) *PrometheusExporter {
	e := &PrometheusExporter{
		logger:   logger,
		registry: registry,
		gauges:   make(map[string]*prometheus.GaugeVec),
		metrics:  make(map[NodeMetricKey]prometheus.Gauge),
	}

	e.ConnectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "opcua_gateway_connections_active",
		Help: "Number of active OPC-UA connections",
	})
	e.SubscriptionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "opcua_gateway_subscriptions_active",
		Help: "Number of active OPC-UA subscriptions",
	})
	e.ReconnectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "opcua_gateway_reconnections_total",
		Help: "Total number of OPC-UA reconnection attempts",
	})
	e.ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "opcua_gateway_errors_total",
		Help: "Total number of errors by type",
	}, []string{"type"})

	registry.MustRegister(e.ConnectionsActive)
	registry.MustRegister(e.SubscriptionsActive)
	registry.MustRegister(e.ReconnectionsTotal)
	registry.MustRegister(e.ErrorsTotal)

	return e
}

// RegisterNode creates or retrieves a Prometheus gauge for the given OPC-UA node.
func (e *PrometheusExporter) RegisterNode(prefix string, nodeName string, subLabels SubscriptionLabels, nodeLabels NodeLabels) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	metricName := prefix + nodeName
	key := NodeMetricKey{
		Namespace:    subLabels.Namespace,
		Subscription: subLabels.Subscription,
		NodeID:       nodeLabels.NodeID,
	}

	if _, exists := e.metrics[key]; exists {
		return nil // Already registered
	}

	gaugeVec, exists := e.gauges[metricName]
	if !exists {
		gaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricName,
			Help: fmt.Sprintf("OPC-UA node value for %s", nodeName),
		}, []string{"namespace", "subscription", "node_id", "unit", "endpoint"})

		if err := e.registry.Register(gaugeVec); err != nil {
			return fmt.Errorf("registering metric %s: %w", metricName, err)
		}
		e.gauges[metricName] = gaugeVec
	}

	gauge := gaugeVec.With(prometheus.Labels{
		"namespace":    subLabels.Namespace,
		"subscription": subLabels.Subscription,
		"node_id":      nodeLabels.NodeID,
		"unit":         nodeLabels.Unit,
		"endpoint":     subLabels.Endpoint,
	})

	e.metrics[key] = gauge
	e.logger.Debug("registered metric",
		zap.String("name", metricName),
		zap.String("nodeID", nodeLabels.NodeID),
	)

	return nil
}

// UpdateNode sets the value of a Prometheus gauge for a given node.
func (e *PrometheusExporter) UpdateNode(key NodeMetricKey, value float64) {
	e.mu.RLock()
	gauge, exists := e.metrics[key]
	e.mu.RUnlock()

	if !exists {
		e.logger.Warn("metric not found for node", zap.String("nodeID", key.NodeID))
		return
	}

	gauge.Set(value)
}

// UnregisterSubscription removes all metrics for a given subscription.
func (e *PrometheusExporter) UnregisterSubscription(namespace, subscription string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for key := range e.metrics {
		if key.Namespace == namespace && key.Subscription == subscription {
			delete(e.metrics, key)
		}
	}

	e.logger.Info("unregistered metrics for subscription",
		zap.String("namespace", namespace),
		zap.String("subscription", subscription),
	)
}
