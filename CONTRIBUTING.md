# Contributing to opcua-kube-gateway

Welcome — and thank you for considering a contribution. This project bridges industrial OPC-UA systems to Kubernetes-native observability, and we welcome contributions from both the Kubernetes and industrial automation communities.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Local Development Setup](#local-development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Running Tests](#running-tests)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Reporting Bugs](#reporting-bugs)
- [Code of Conduct](#code-of-conduct)

---

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.21+ | Build the operator |
| Docker | 24+ | Run the simulator and build images |
| kubectl | 1.26+ | Interact with Kubernetes |
| [kind](https://kind.sigs.k8s.io) or [k3d](https://k3d.io) | latest | Local Kubernetes cluster |
| make | any | Build automation |

Optional but useful:
- `controller-gen` — auto-installed by `make generate`
- `golangci-lint` — auto-installed by `make lint`
- Helm 3 — for testing the chart

---

## Local Development Setup

### 1. Clone the repo

```bash
git clone https://github.com/opcua-kube-gateway/opcua-kube-gateway.git
cd opcua-kube-gateway
```

### 2. Start the OPC-UA simulator

The simulator runs a realistic OPC-UA server with four nodes (Temperature, Pressure, MachineStatus, ProductionCount). You don't need a real PLC to develop locally.

```bash
docker compose up -d simulator
```

Verify it's running:

```bash
docker compose logs simulator
# Should show:
# OPC-UA Simulator running at opc.tcp://0.0.0.0:4840
# Nodes:
#   ns=2;s=Temperature   (Double, sine wave 20-80°C)
#   ns=2;s=Pressure      (Double, random walk 1-10 bar)
#   ns=2;s=MachineStatus (Int32, cycles 0/1/2)
#   ns=2;s=ProductionCount (Int64, counter)
```

### 3. Build the operator

```bash
make build
# Binary written to ./bin/operator
```

### 4. Run the operator locally

The operator can run outside the cluster while still managing resources in a local kind/k3d cluster.

```bash
# Create a local cluster (if you don't have one)
kind create cluster

# Install the CRD
make generate
kubectl apply -f charts/opcua-kube-gateway/crds/

# Run the operator (connects to your current kubeconfig context)
./bin/operator \
  --metrics-bind-address=:8080 \
  --health-probe-bind-address=:8081
```

### 5. Apply a test subscription

```bash
kubectl apply -f examples/simulator-test.yaml

# Watch the status
kubectl get opcuasubscriptions -w
```

### 6. Query metrics

```bash
curl -s http://localhost:8080/metrics | grep opcua_
```

---

## Project Structure

```
opcua-kube-gateway/
├── api/
│   └── v1alpha1/
│       ├── types.go                  # CRD types: OPCUASubscription spec + status
│       ├── groupversion_info.go      # API group registration
│       └── zz_generated.deepcopy.go  # Auto-generated — do not edit
├── cmd/
│   └── operator/
│       └── main.go                   # Entrypoint: sets up manager, controller, exporter
├── internal/
│   ├── controller/
│   │   └── opcuasubscription_controller.go  # Reconcile loop — core controller logic
│   ├── opcua/
│   │   └── client.go                 # OPC-UA client: connect, subscribe, data change callbacks
│   └── exporter/
│       └── prometheus.go             # Prometheus metrics registration and updates
├── simulator/
│   ├── server.py                     # Python OPC-UA server (asyncua)
│   ├── Dockerfile
│   └── requirements.txt
├── charts/
│   └── opcua-kube-gateway/           # Helm chart
├── examples/
│   ├── pump-monitoring.yaml          # Real-world example
│   └── simulator-test.yaml           # Development/CI example
├── hack/
│   └── boilerplate.go.txt            # License header for generated files
├── Makefile                          # Build, test, lint, generate targets
└── docker-compose.yaml               # Simulator for local dev
```

### Key entry points

- **Adding a new exporter** (e.g., MQTT, InfluxDB): model it after `internal/exporter/prometheus.go` and wire it up in the controller
- **Changing the CRD API**: edit `api/v1alpha1/types.go`, then run `make generate`
- **Changing reconnect / subscription behaviour**: `internal/opcua/client.go`
- **Changing reconcile logic**: `internal/controller/opcuasubscription_controller.go`

---

## Making Changes

### API changes

If you modify anything in `api/v1alpha1/types.go`, regenerate the CRD manifests and deepcopy code:

```bash
make generate
```

Commit the generated files (`zz_generated.deepcopy.go` and `charts/.../crds/`) together with your change.

### Code formatting

```bash
make fmt      # gofmt
make vet      # go vet
make lint     # golangci-lint (installs automatically if missing)
```

### Adding a new OPC-UA feature

If you're working on OPC-UA protocol features and want to test against a real server, note the server type and security mode in your PR. We accept contributions tested against:
- The included simulator (minimum requirement)
- Prosys OPC-UA Simulation Server (free for evaluation)
- Unified Automation C++ Demo Server
- Real industrial hardware (Siemens S7, Beckhoff, etc.)

---

## Running Tests

```bash
make test
```

Tests run with race detection and require 80% coverage. The CI environment runs the simulator as a service — locally, ensure it's running via `docker compose up -d simulator` before running integration tests.

Coverage report:

```bash
go tool cover -html=coverage.out
```

---

## Submitting a Pull Request

1. Fork the repo and create a branch: `git checkout -b feat/my-feature`
2. Make your changes with tests
3. Ensure `make lint` and `make test` pass
4. Open a PR against `main`

**In your PR description, include:**
- What the change does and why
- How you tested it (simulator / real hardware / unit tests only)
- Any OPC-UA server type used for testing, if relevant

We aim to review PRs within 48 hours. First-time contributors: your first PR will get extra attention — we'll help you through the process.

---

## Reporting Bugs

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md). The most useful information for debugging OPC-UA issues is:
- The OPC-UA server type and version
- The security mode in use
- Relevant operator logs (`kubectl logs`)
- The `OPCUASubscription` spec (redact endpoint hostnames if needed)

---

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). Be respectful and constructive. We're here to build something useful together.
