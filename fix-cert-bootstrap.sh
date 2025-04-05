#!/bin/bash

set -e

NAMESPACE="secrethor-system"
ISSUER_FILE="config/webhook/certmanager/issuer.yaml"
CERT_FILE="config/webhook/certmanager/certificate.yaml"

echo "📌 Etapa 1: Garantindo label para ignorar validação no namespace do operator"
kubectl label namespace $NAMESPACE cert-manager.io/disable-validation=true --overwrite

echo "🧹 Etapa 2: Limpando recursos antigos (certificaterequest, certificate, secret)..."
kubectl delete certificaterequest -n $NAMESPACE --all --ignore-not-found
kubectl delete certificate secrethor-webhook-cert -n $NAMESPACE --ignore-not-found
kubectl delete secret webhook-server-cert -n $NAMESPACE --ignore-not-found

echo "📄 Etapa 3: Reaplicando issuer e certificate"
kubectl apply -f "$ISSUER_FILE"
kubectl apply -f "$CERT_FILE"

echo "⏳ Etapa 4: Aguardando Secret webhook-server-cert ser criada..."
for i in {1..20}; do
    if kubectl get secret webhook-server-cert -n $NAMESPACE > /dev/null 2>&1; then
        echo "✅ Secret webhook-server-cert criada com sucesso!"
        exit 0
    fi
    echo "⏱️  Aguardando... ($i)"
    sleep 3
done

echo "❌ Timeout: Secret webhook-server-cert não foi criada. Verifique os logs do cert-manager."
exit 1
