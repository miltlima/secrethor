# Secrethor Helm Chart 🔒

Secrethor is a Kubernetes operator that audits the hygiene of Secrets. It reports Secrets that are stale, unused or out of standard through the `SecretPolicy` status and Prometheus metrics. It never modifies a Secret, and it only blocks one when a policy asks for it.

## Features

- ✅ Read-only audit of Secrets against `SecretPolicy` rules
- ✅ Detection of expired (`maxAgeDays`) and unused (`detectUnused`) Secrets
- ✅ Type, required key, forbidden key and value rules
- ✅ Findings in `kubectl get secretpolicies` and in Prometheus metrics
- ✅ Optional admission webhook: warnings in `Audit` mode, rejection in `Enforce` mode

## Installation

```bash
helm repo add secrethor https://miltlima.github.io/secrethor

helm install secrethor secrethor/secrethor \
  --namespace secrethor-system \
  --create-namespace
```

This installs the auditor only: nothing is blocked and cert-manager is not needed.

### Optional: admission webhook

Requires cert-manager in the cluster.

```bash
helm upgrade secrethor secrethor/secrethor \
  --namespace secrethor-system \
  --set webhook.enabled=true
```

### Values

| Value | Default | Description |
|---|---|---|
| `audit.interval` | `15m` | How often each policy is re-audited |
| `audit.maxFindings` | `100` | Findings kept in the status of each policy |
| `webhook.enabled` | `false` | Serve the admission webhook (requires cert-manager) |
| `webhook.failurePolicy` | `Fail` | What the API server does when the webhook is unreachable |
| `namespace.create` | `false` | Create the namespace from the chart |

### Upgrading

Helm does not upgrade CRDs. Apply the CRD in `crds/` with `kubectl apply` before upgrading the chart. Existing policies become `enforcement: Audit`; set `enforcement: Enforce` on the ones that must keep rejecting Secrets.

## Uninstall 
```bash
helm uninstall secrethor --namespace secrethor-system
```