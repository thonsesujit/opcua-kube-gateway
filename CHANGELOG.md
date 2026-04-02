# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.1.0] - 2026-04-02

Initial release of `opcua-kube-gateway`.

### Added

**Operator**
- `OPCUASubscription` CRD (`opcua.gateway.io/v1alpha1`) for declaring OPC-UA subscriptions as Kubernetes resources
- Kubernetes controller with finalizer-based clean teardown
- OPC-UA client using MonitoredItem subscriptions (server-push, not polling)
- Security mode support: `None` (default); `Sign` and `SignAndEncrypt` in spec (full implementation planned for v0.2.0)
- Per-subscription and per-node status in `.status` subresource
- Kubernetes Events emitted on connection lifecycle transitions (`Connecting`, `Connected`, `Error`, `Disconnected`)
- Leader election for safe multi-replica deployments

**Prometheus Exporter**
- Dynamic metric registration per node at reconcile time
- Metric naming: `<prefix><node_name>` with labels `{namespace, subscription, node_id, unit}`
- Built-in operator metrics: active connections, active subscriptions, error counters

**Helm Chart**
- Full Helm chart under `charts/opcua-kube-gateway`
- Optional `ServiceMonitor` creation for Prometheus Operator
- Optional Grafana dashboard `ConfigMap` creation
- Non-root, read-only filesystem, dropped capabilities security context by default

**Simulator**
- Dockerised OPC-UA simulator (`simulator/`) using [asyncua](https://github.com/FreeOpcUa/opcua-asyncio)
- Four realistic industrial nodes: Temperature (sine wave), Pressure (random walk), MachineStatus (state machine), ProductionCount (counter)
- Used in CI and local development — no real PLC required

**CI/CD**
- GitHub Actions CI: lint, test (with simulator), build
- 80% coverage gate with race detection
- Release workflow for image publishing to `ghcr.io`

**Examples**
- `examples/pump-monitoring.yaml` — real-world pump monitoring subscription
- `examples/simulator-test.yaml` — development and testing subscription

[Unreleased]: https://github.com/opcua-kube-gateway/opcua-kube-gateway/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/opcua-kube-gateway/opcua-kube-gateway/releases/tag/v0.1.0
