---
name: Bug report
about: Something is not working correctly
labels: bug
---

## Describe the bug

<!-- A clear and concise description of what went wrong. -->

## Environment

| Field | Value |
|---|---|
| opcua-kube-gateway version | |
| Kubernetes version | |
| Helm chart version (if applicable) | |
| OPC-UA server type | <!-- e.g. Siemens S7-1200, Kepware KEPServerEX, Prosys, custom --> |
| OPC-UA server version | |
| Security mode | <!-- None / Sign / SignAndEncrypt --> |

## OPCUASubscription spec

```yaml
# Paste your OPCUASubscription YAML here.
# Redact the endpoint hostname if needed (e.g. opc.tcp://[redacted]:4840).
```

## What happened

<!-- What did you observe? -->

## What you expected

<!-- What did you expect to happen? -->

## Operator logs

```
# kubectl logs -n <namespace> deployment/opcua-kube-gateway
```

## Subscription status

```
# kubectl describe opcuasubscription <name>
```

## Additional context

<!-- Screenshots, Grafana dashboards, or anything else that helps. -->
