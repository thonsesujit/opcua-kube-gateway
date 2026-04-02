# opcua-kube-gateway

A Kubernetes-native OPC-UA gateway that bridges factory-floor industrial sensors to cloud-native observability stacks (Prometheus, Grafana).

## Architecture

```
Factory Floor                     Kubernetes Cluster
┌────────────────┐               ┌─────────────────────────────┐
│ OPC-UA Server  │               │  opcua-kube-gateway Pod     │
│ (PLC/SCADA)    │◀──opc.tcp──▶ │  ┌───────────────────────┐  │
└────────────────┘               │  │ CRD Watcher           │  │
                                 │  │ OPC-UA Subscriber     │  │
                                 │  │ Prometheus Exporter   │  │
                                 │  └───────────┬───────────┘  │
                                 └──────────────┼──────────────┘
                          ┌─────────────────────┼──────────┐
                          ▼                     ▼          ▼
                   ┌────────────┐     ┌──────────────┐  ┌───────┐
                   │ Prometheus │────▶│ Grafana      │  │ CRD   │
                   └────────────┘     │ Dashboard    │  │.status│
                                      └──────────────┘  └───────┘
```

## Quick Start

### Prerequisites

- Kubernetes cluster (1.26+)
- Helm 3
- An OPC-UA server (or use the included simulator)

### Install

```bash
helm install opcua-gateway ./charts/opcua-kube-gateway
```

### Create a subscription

```yaml
apiVersion: opcua.gateway.io/v1alpha1
kind: OPCUASubscription
metadata:
  name: pump-monitoring
  namespace: factory-floor
spec:
  endpoint: opc.tcp://plc-01.factory:4840
  securityMode: None
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
```

```bash
kubectl apply -f subscription.yaml
```

### Check status

```bash
kubectl get opcuasubscriptions
```

```
NAME              ENDPOINT                     PHASE       AGE
pump-monitoring   opc.tcp://plc-01.factory:4840   Connected   2m
```

### Query metrics

```promql
factory_pump_temperature{subscription="pump-monitoring"}
```

## Development

### Local setup with simulator

```bash
# Start the OPC-UA simulator
docker compose up -d simulator

# Run the operator locally
make build
./bin/operator --metrics-bind-address=:8080 --health-probe-bind-address=:8081
```

### Run tests

```bash
make test
```

### Lint

```bash
make lint
```

### Generate CRD manifests

```bash
make generate
```

## CRD Reference

### OPCUASubscription Spec

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `endpoint` | string | yes | - | OPC-UA server URL |
| `securityMode` | enum | no | `None` | `None`, `Sign`, or `SignAndEncrypt` |
| `nodes` | array | yes | - | List of OPC-UA nodes to subscribe to |
| `exporters.prometheus.enabled` | bool | no | `true` | Enable Prometheus metrics |
| `exporters.prometheus.prefix` | string | no | `opcua_` | Metric name prefix |

### Node Spec

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `nodeId` | string | yes | - | OPC-UA node ID (e.g., `ns=2;s=Temperature`) |
| `name` | string | yes | - | Metric name (lowercase, underscores) |
| `unit` | string | no | - | Unit label for Prometheus |
| `interval` | duration | no | `5s` | Subscription publishing interval |

### Status

| Field | Type | Description |
|---|---|---|
| `phase` | string | `Connecting`, `Connected`, `Error`, `Disconnected` |
| `lastConnected` | timestamp | Last successful connection time |
| `message` | string | Human-readable status message |
| `conditions` | array | Standard Kubernetes conditions |
| `nodes` | array | Per-node status with last value |

## Helm Values

See [values.yaml](charts/opcua-kube-gateway/values.yaml) for all configurable options.

Key settings:

| Value | Default | Description |
|---|---|---|
| `image.repository` | `ghcr.io/opcua-kube-gateway/opcua-kube-gateway` | Container image |
| `leaderElection.enabled` | `true` | Enable leader election |
| `serviceMonitor.enabled` | `false` | Create Prometheus ServiceMonitor |
| `grafanaDashboard.enabled` | `false` | Create Grafana dashboard ConfigMap |
| `resources.limits.memory` | `256Mi` | Memory limit |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup, project structure, and how to submit a pull request.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features and milestones.

## Architecture

See [docs/architecture.md](docs/architecture.md) for a deep-dive into the reconcile loop, component design, and key decisions.

## License

Apache 2.0
