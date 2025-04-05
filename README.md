<p align="center">
  <img src="assets/secrethor-logo.jpg" alt="Secrethor Logo" width="300"/>
</p>

<p align="center">
  <a href="https://opensource.org/licenses/MIT"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg"/></a>
  <a href="https://github.com/miltlima/secrethor/actions/workflows/test.yaml"><img alt="Build Status" src="https://github.com/miltlima/secrethor/actions/workflows/test.yaml/badge.svg"/></a>
</p>

# Secrethor

**Secrethor** is a Kubernetes Operator that manages the lifecycle of secrets across your cluster, enforcing security best practices, governance, and compliance.

It allows you to define declarative policies to detect expired, unused (orphaned), or mislocated secrets, helping DevOps and SRE teams maintain visibility and control over sensitive assets.

---

## ✨ Features

- 🔍 Automatic discovery of all Kubernetes `Secrets`
- ✅ Declarative `SecretPolicy` CRDs to enforce:
  - Maximum age (`maxAgeDays`)
  - Namespace restrictions (`allowedNamespaces`)
  - Allowed Secret types (`allowedTypes`)
  - Required and forbidden keys (`requiredKeys`, `forbiddenKeys`)
  - Regex and content-based validation (`valueConstraints`)
- 🧠 Built-in webhook for admission control
- 📜 Policy violation logs for visibility
- 📦 Built in Go using Operator SDK (extensible and maintainable)

---

## 🔧 Example `SecretPolicy`

```yaml
apiVersion: secrets.secrethor.dev/v1alpha1
kind: SecretPolicy
metadata:
  name: secure-policy
spec:
  allowedNamespaces:
  - default
  - prod
  - staging
  maxAgeDays: 30
  allowedTypes:
  - Opaque
  - kubernetes.io/basic-auth
  - kubernetes.io/dockerconfigjson

  requiredKeys:
  - username
  - password

  forbiddenKeys:
  - token
  - privateKey

  valueConstraints:
    password:
      minLength: 12
      mustContain:
      - upper
      - lower
      - number
      - special
    username:
      minLength: 4
      regex: "^[a-zA-Z0-9_.-]+$"

```

---

## 🚀 Getting Started

### Prerequisites

- `go` version v1.22.0+
- `docker` version 17.03+
- `kubectl` version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- `cert-manager` installed on the cluster

---

## 🔐 What is `allowedNamespaces`?

The `allowedNamespaces` field defines which Kubernetes namespaces are authorized to contain Secrets under your organization’s policy.

### Why use it?

- 🚫 Prevents sensitive secrets from being created in non-secure namespaces
- 🛡 Encourages security best practices and namespace segmentation
- ✅ Helps ensure compliance with standards like PCI, SOC2, ISO, GDPR

If a Secret is created in a namespace not listed in `allowedNamespaces`, Secrethor will deny the request.

---

## 🗺 Roadmap

- ✅ Webhook for policy enforcement
- ✅ Namespace policy enforcement
- ✅ Secret type validation
- ✅ Key/content validation (length, pattern, etc.)
- 🔜 Expired secrets detection
- 🔜 Unused secret detection
- 🔜 Secret rotation support (Vault, AWS Secrets Manager, SOPS)
- 🔜 Prometheus metrics & Grafana dashboards
- 🔜 Slack/MS Teams alert integration
- 🔜 OLM/OperatorHub support via Helm Chart

---

## 🤝 Contributing

Contributions are welcome!

If you want to contribute new features, improve documentation, or report a bug — feel free to open an issue or submit a PR.

---

## 🪪 License

This project is licensed under the [MIT License](https://opensource.org/licenses/MIT).

---

## 👤 Author

**Milton Lima de Jesus**  
DevOps / SRE Engineer  
[linkedin.com/in/miltonlimaj](https://linkedin.com/in/miltonlimaj)