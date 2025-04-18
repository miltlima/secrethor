# Secrethor Helm Chart 🔒

Secrethor is a Kubernetes Operator for managing and validating Secrets, with support for policies, custom CRDs (`SecretPolicy`), and a complementary CLI (`secrethor-cli`).

## Features

- ✅ Admission webhook for Secret validation
- ✅ SecretPolicy CRD with rules like allowed types, required keys, forbidden keys
- ✅ Namespace scoping and maxAgeDays support
- ✅ Integration with cert-manager for TLS
- ✅ Built with [operator-sdk](https://sdk.operatorframework.io/)
- ✅ Helm-native installation

## Installation

### Two-step deployment (recommended)

# Step 1: Install without the webhook

```bash
helm repo add secrethor https://miltlima.github.io/secrethor 
```

```bash
helm install secrethor secrethor/secrethor \
  --namespace secrethor-system \
  --create-namespace \
  --set webhook.enabled=false
```

# Step 2: Enable webhook once pods/services are ready
```bash
helm upgrade secrethor secrethor/secrethor \
  --namespace secrethor-system \
  --set webhook.enabled=true
```

# Optional: Create namespace via Helm
```yaml
namespace:
  create: true
```

## Uninstall 
```bash
helm uninstall secrethor --namespace secrethor-system
kubectl delete validatingwebhookconfiguration secrets.secrethor.dev --ignore-not-found
```