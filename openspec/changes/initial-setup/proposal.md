# Proposal: opcua-kube-gateway

## Summary

Build a Kubernetes-native OPC-UA gateway that bridges factory-floor industrial sensors to cloud-native observability stacks (Prometheus, OpenTelemetry, Grafana).

## Problem

Industrial manufacturing runs on OPC-UA for sensor data, while modern infrastructure runs on Kubernetes with Prometheus/OpenTelemetry for observability. There is **no existing Kubernetes operator** that bridges these two worlds. Engineers must build custom glue code for every factory-to-cloud integration.

Companies like Siemens, ABB, BMW, and defense contractors all face this exact integration pain point — they run K8s in IT and OPC-UA on the factory floor, with no standardized bridge between them.

## Solution

A Go-based Kubernetes operator that:
- Watches `OPCUASubscription` Custom Resource Definitions (CRDs)
- Connects to OPC-UA servers and subscribes to data nodes
- Exports industrial sensor data as Prometheus metrics and OpenTelemetry signals
- Includes pre-built Grafana dashboards for factory monitoring
- Deploys via Helm chart with production-ready defaults

## Architecture

```
┌─────────────────────────┐         ┌──────────────────────────────┐
│  OPC-UA Server          │         │  opcua-kube-gateway Pod       │
│  (PLC / Simulator)      │◀───────▶│                              │
│  opc.tcp://plc:4840     │  OPC-UA │  ┌──────────────────────┐   │
└─────────────────────────┘         │  │ CRD Watcher          │   │
                                    │  │ (K8s controller)     │   │
                                    │  └──────────┬───────────┘   │
                                    │             │               │
                                    │  ┌──────────▼───────────┐   │
                                    │  │ OPC-UA Subscriber    │   │
                                    │  │ (browse + subscribe) │   │
                                    │  └──────────┬───────────┘   │
                                    │             │               │
                                    │  ┌──────────▼───────────┐   │
                                    │  │ Exporters            │   │
                                    │  │ • Prometheus /metrics │   │
                                    │  │ • OTLP exporter      │   │
                                    │  │ • CRD .status update │   │
                                    │  └──────────────────────┘   │
                                    └──────────────────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────┐
                    ▼                         ▼                 ▼
            ┌──────────────┐      ┌───────────────┐    ┌──────────────┐
            │ Prometheus   │      │ OpenTelemetry  │    │ K8s CRD      │
            │              │      │ Collector      │    │ .status      │
            └──────┬───────┘      └───────────────┘    └──────────────┘
                   │
            ┌──────▼───────┐
            │ Grafana      │
            │ Dashboard    │
            └──────────────┘
```

## CRD Example

```yaml
apiVersion: opcua.gateway.io/v1alpha1
kind: OPCUASubscription
metadata:
  name: pump-monitoring
  namespace: factory-floor
spec:
  endpoint: opc.tcp://plc-01.factory:4840
  securityMode: SignAndEncrypt
  nodes:
    - nodeId: "ns=2;s=Temperature"
      name: pump_temperature
      unit: celsius
      interval: 1s
    - nodeId: "ns=2;s=Pressure"
      name: pump_pressure
      unit: bar
      interval: 5s
  exporters:
    prometheus:
      enabled: true
      prefix: factory_
    opentelemetry:
      enabled: true
      endpoint: otel-collector:4317
```

## Tech Stack

- **Language:** Go
- **K8s:** controller-runtime (Operator SDK)
- **OPC-UA:** gopcua/opcua library
- **Observability:** Prometheus client, OpenTelemetry SDK
- **Deployment:** Helm chart, Docker, GitHub Actions CI/CD
- **Testing:** Go testing + envtest (K8s integration tests)

## MVP Scope

1. CRD definition (`OPCUASubscription`) with validation
2. Operator that watches CRDs and manages OPC-UA connections
3. Prometheus metrics exporter (/metrics endpoint)
4. Basic Grafana dashboard (included in Helm chart)
5. Helm chart for deployment
6. OPC-UA simulator for local development and testing
7. GitHub Actions CI/CD pipeline
8. README with architecture diagram, quick start, and examples

## Non-Goals (MVP)

- OpenTelemetry exporter (Phase 2)
- OPC-UA security certificate management (Phase 2)
- Multi-cluster support (Phase 2)
- Historical data backfill (Phase 2)
- Web UI (Phase 2)

## Why This Project Is Unique

There is no production-quality Kubernetes operator that bridges OPC-UA to cloud-native observability. Existing projects (open62541, node-opcua, eclipse-milo) are libraries, not operators. Only someone with real PLC/OPC-UA experience AND K8s operator skills can design the CRDs correctly — understanding OPC-UA namespaces, node IDs, security modes, and subscription semantics.

## Target Employers This Impresses

- **Siemens, ABB, BMW** — they run K8s + OPC-UA and this solves their integration pain
- **Helsing, HENSOLDT** — sensor data ingestion from heterogeneous sources
- **Any IIoT company** — universal OT-to-IT bridge

## Estimated Effort

6–8 weekends (~60–80 hours total for MVP)

## Success Criteria

- [ ] Helm install deploys the operator and it watches for CRDs
- [ ] Creating an OPCUASubscription CRD connects to an OPC-UA server and starts exporting metrics
- [ ] Prometheus can scrape factory sensor data from the operator
- [ ] Grafana dashboard shows live industrial data
- [ ] CI/CD pipeline passes (lint, test, build, release)
- [ ] Test coverage > 80%
- [ ] README with architecture diagram, quick start, and GIF demo
