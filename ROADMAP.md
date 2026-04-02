# Roadmap

This is the public roadmap for `opcua-kube-gateway`. Items are grouped by milestone. Checked items are complete; unchecked items are planned or in progress.

Have a use case not covered here? [Open a feature request](https://github.com/opcua-kube-gateway/opcua-kube-gateway/issues/new?template=feature_request.md).

---

## v0.1.0 — Core Foundation ✓ (current)

The minimum viable operator: declare an OPC-UA subscription as a Kubernetes resource, get Prometheus metrics.

- [x] `OPCUASubscription` CRD with spec validation
- [x] Kubernetes controller with finalizer-based teardown
- [x] OPC-UA client using MonitoredItem subscriptions (push, not poll)
- [x] Prometheus exporter with dynamic metric registration
- [x] Per-node status in `.status.nodes`
- [x] Kubernetes Events for connection lifecycle
- [x] Helm chart with ServiceMonitor and Grafana dashboard toggles
- [x] OPC-UA simulator for local development and CI
- [x] CI with 80% coverage gate and race detection

---

## v0.2.0 — Security & Authentication

Production deployments require secure OPC-UA connections. This milestone makes the gateway usable in environments where security mode `None` is not acceptable.

- [ ] OPC-UA `Sign` security mode
- [ ] OPC-UA `SignAndEncrypt` security mode
- [ ] Client certificate management via Kubernetes Secrets
- [ ] Certificate auto-rotation support
- [ ] Username/password authentication (`UserNameIdentityToken`)
- [ ] TLS configuration for the Prometheus metrics endpoint

---

## v0.3.0 — Additional Export Targets

Prometheus is not always the right sink. This milestone adds exporters for common industrial and cloud-native data pipelines.

- [ ] MQTT exporter (publish node values to MQTT broker)
- [ ] InfluxDB v2 exporter
- [ ] Kafka exporter (node values as CloudEvents)
- [ ] Configurable export strategy per node (not all nodes need all exporters)

---

## v0.4.0 — Observability & Operations

Make the gateway itself observable and easier to operate in production.

- [ ] Bundled Grafana dashboard (connection health, subscription counts, error rates)
- [ ] Alert rules (PrometheusRule) for connection loss and error spikes
- [ ] Improved reconnection with exponential backoff
- [ ] Spec-diff on reconcile (reconnect only when endpoint/security changes)
- [ ] Per-node error counters in Prometheus

---

## v0.5.0 — Scale & Resilience

For deployments with many subscriptions or strict reliability requirements.

- [ ] Multi-endpoint failover (primary + backup OPC-UA server)
- [ ] Configurable reconnect backoff and retry limits
- [ ] OPC-UA Historical Data Access (read historical values on startup)
- [ ] Rate limiting for high-frequency nodes
- [ ] Horizontal scaling with sharded subscriptions

---

## Beyond (ideas, not committed)

- [ ] Edge deployment profile (low-resource mode for Raspberry Pi / industrial gateways)
- [ ] Multi-cluster support (subscriptions spanning multiple clusters)
- [ ] Web UI for browsing OPC-UA node trees and creating subscriptions
- [ ] OPC-UA UA-over-HTTPS transport
- [ ] CNCF Sandbox proposal

---

## How to Influence the Roadmap

- **Vote** on issues with a 👍 reaction — high-vote items get prioritized
- **Comment** on roadmap items with your use case
- **Contribute** — if something matters to you, we'll help you build it

---

*Last updated: April 2026*
