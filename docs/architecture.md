# Architecture

This document describes the internal design of `opcua-kube-gateway` — how components fit together, how data flows through the system, and why key decisions were made.

---

## System Overview

```
Factory Floor                        Kubernetes Cluster
──────────────────                   ──────────────────────────────────────
                                     ┌─────────────────────────────────┐
OPC-UA Server                        │  opcua-kube-gateway Pod         │
(PLC / SCADA / DCS)                  │                                 │
                                     │  ┌─────────────────────────┐   │
  opc.tcp://plc:4840  ◀──────────────│──│  OPC-UA Client          │   │
                                     │  │  (internal/opcua)       │   │
                                     │  └──────────┬──────────────┘   │
                                     │             │ DataChange        │
                                     │  ┌──────────▼──────────────┐   │
                                     │  │  Reconciler             │   │
                                     │  │  (internal/controller)  │   │
                                     │  └──────────┬──────────────┘   │
                                     │             │ UpdateNode        │
                                     │  ┌──────────▼──────────────┐   │
                                     │  │  Prometheus Exporter    │   │
                                     │  │  (internal/exporter)    │   │
                                     │  └──────────┬──────────────┘   │
                                     └─────────────┼───────────────────┘
                                                   │ /metrics
                                     ┌─────────────▼──────────┐
                                     │  Prometheus             │
                                     └─────────────┬───────────┘
                                                   │
                                     ┌─────────────▼───────────┐
                                     │  Grafana Dashboard      │
                                     └─────────────────────────┘

Kubernetes API
──────────────
  OPCUASubscription CRD ◀──── kubectl apply -f subscription.yaml
       │ watch
  Reconciler (controller-runtime)
```

---

## Components

### 1. CRD — `OPCUASubscription` (`api/v1alpha1/`)

The `OPCUASubscription` custom resource is the user-facing API. Users declare what OPC-UA nodes they want to subscribe to and how to export the data.

**Spec** — desired state:
- `endpoint` — the OPC-UA server URL (`opc.tcp://...`)
- `securityMode` — `None`, `Sign`, or `SignAndEncrypt`
- `nodes[]` — list of node IDs, metric names, units, and intervals
- `exporters.prometheus` — prefix and enable/disable flag

**Status** — observed state:
- `phase` — `Connecting`, `Connected`, `Error`, `Disconnected`
- `conditions` — standard Kubernetes condition array
- `nodes[]` — per-node last value and timestamp
- `lastConnected` — timestamp of last successful connection

### 2. Controller (`internal/controller/`)

Implements the Kubernetes controller pattern using [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime). Watches `OPCUASubscription` resources and drives them toward desired state.

**Key responsibilities:**
- Add/remove finalizers for clean teardown
- Start/stop OPC-UA client goroutines
- Update `.status` based on connection events
- Emit Kubernetes Events for significant transitions
- Register Prometheus metrics per node

**Runtime state** is held in `states map[string]*subscriptionState` (keyed by `namespace/name`), protected by a mutex. This tracks the active OPC-UA client and its cancel function for each subscription.

### 3. OPC-UA Client (`internal/opcua/`)

Wraps [gopcua](https://github.com/gopcua/opcua) to manage the OPC-UA session lifecycle.

**Responsibilities:**
- Establish and maintain the OPC-UA connection
- Create monitored items for each configured node
- Invoke a `DataChange` callback on every received value
- Handle reconnection on transient failures

**Node subscription** uses OPC-UA's `MonitoredItem` mechanism — the server pushes value changes to the client at the configured publishing interval. This is more efficient than polling.

### 4. Prometheus Exporter (`internal/exporter/`)

Manages a set of `prometheus.Gauge` metrics, one per subscribed node.

**Key design:** metrics are registered dynamically at runtime when a subscription is reconciled, not statically at startup. This allows any number of `OPCUASubscription` resources to coexist, each with their own metric namespace and prefix.

**Metric naming:** `<prefix><node_name>` with labels `{namespace, subscription, node_id, unit}`.

Example:
```
factory_pump_temperature{namespace="prod", subscription="pump-monitoring", node_id="ns=2;s=Temperature", unit="celsius"} 47.3
```

---

## Reconcile Loop

The reconcile loop is the heart of the controller. It is called by controller-runtime whenever an `OPCUASubscription` is created, updated, or deleted — and on periodic re-queue.

```
Reconcile(req) called
        │
        ▼
  Fetch OPCUASubscription
        │
        ├─ Not found ──▶ return (deleted, nothing to do)
        │
        ▼
  DeletionTimestamp set?
        │
        ├─ Yes ──▶ handleDeletion()
        │           Stop goroutine
        │           Unregister metrics
        │           Remove finalizer
        │           return
        │
        ▼
  Finalizer present?
        │
        ├─ No ──▶ Add finalizer, Update, return
        │
        ▼
  reconcileSubscription()
        │
        ├─ Register Prometheus metrics for all nodes
        │
        ├─ Is subscription already running?
        │   └─ Yes ──▶ Stop it (restart on any spec change, MVP behaviour)
        │
        ├─ Build OPC-UA client config from spec
        │
        ├─ Start goroutine: runSubscription()
        │   └─ Connect to OPC-UA server
        │   └─ Subscribe to nodes
        │   └─ On DataChange: update Prometheus gauge
        │   └─ On error: update status to Error
        │
        ├─ Update status to Connecting
        │
        └─ return (no requeue — goroutine drives state forward)
```

**Note:** The current implementation restarts the OPC-UA connection on any spec change. A future improvement (tracked in the roadmap) would diff the spec and only reconnect when the endpoint or security mode changes.

---

## OPC-UA Connection Lifecycle

```
  kubectl apply
       │
       ▼
  Reconciler starts goroutine
       │
       ▼
  opcClient.Connect(ctx)
       │
       ├─ Dial opc.tcp endpoint
       ├─ Negotiate security
       ├─ Create session
       ├─ Activate session
       │
       ▼
  Create Subscription on server
       │
       ▼
  Add MonitoredItems (one per node)
       │
       ▼
  Publish loop (server pushes changes)
       │
       ├─ DataChange received
       │   └─ callback → Exporter.UpdateNode()
       │
       ├─ Context cancelled (kubectl delete / spec change)
       │   └─ Close session cleanly
       │
       └─ Connection error
           └─ Update status to Error
           └─ Goroutine exits (future: retry with backoff)
```

---

## Security Model

Security is configured per `OPCUASubscription` via `spec.securityMode`:

| Mode | Description |
|---|---|
| `None` | No security (plaintext). Suitable for isolated networks / development. |
| `Sign` | Messages are signed with certificates. Not yet fully implemented. |
| `SignAndEncrypt` | Messages are signed and encrypted. Not yet fully implemented. |

Certificate-based security (`Sign`, `SignAndEncrypt`) is on the roadmap for v0.2.0 and will use Kubernetes Secrets to store the client certificate and private key.

---

## Simulator

The simulator (`simulator/server.py`) is a Python OPC-UA server using [asyncua](https://github.com/FreeOpcUa/opcua-asyncio). It runs as a Docker container and is used for:
- Local development (no real hardware needed)
- CI integration tests (see `.github/workflows/ci.yaml`)

Nodes exposed:

| Node ID | Type | Behaviour |
|---|---|---|
| `ns=2;s=Temperature` | Double | Sine wave 20–80°C, 60s period |
| `ns=2;s=Pressure` | Double | Bounded random walk 1–10 bar |
| `ns=2;s=MachineStatus` | Int32 | Cycles: Off(0), Running(1×7), Error(2), Running(1) |
| `ns=2;s=ProductionCount` | Int64 | Monotonically increasing counter |

---

## Design Decisions

**Why a Kubernetes operator (not a sidecar or DaemonSet)?**
The operator pattern allows OPC-UA subscriptions to be declared as Kubernetes resources — versioned, namespaced, auditable, and manageable with standard tools. Users don't need to manage separate configuration files or agents.

**Why Prometheus as the first exporter?**
Prometheus is the de facto standard in the Kubernetes ecosystem. Most teams already have it. It requires no additional infrastructure to get value immediately.

**Why Go and gopcua?**
The operator ecosystem is Go-native (controller-runtime, kubebuilder). `gopcua` is the most actively maintained OPC-UA library for Go and covers the subset of the specification we need.

**Why restart the connection on any spec change (current MVP)?**
Simplicity. Diffing the spec to determine what changed and applying incremental changes to a live OPC-UA session is complex and error-prone. Restart-on-change is safe and predictable. The overhead is acceptable for industrial subscription intervals (seconds, not milliseconds).
