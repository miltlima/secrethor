package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/recorder"

	"github.com/prometheus/client_golang/prometheus"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

var (
	expiredSecrets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "secrethor_expired_secrets_total",
			Help: "Total number of expired secrets detected",
		},
		[]string{"namespace", "name"},
	)
)

func init() {
	prometheus.MustRegister(expiredSecrets)
}

type SecretExpiryChecker struct {
	Client   client.Client
	Recorder recorder.EventRecorder
}

func (c *SecretExpiryChecker) CheckExpiredSecrets(ctx context.Context) error {
	logger := log.FromContext(ctx)

	var secrets corev1.SecretList
	if err := c.Client.List(ctx, &secrets); err != nil {
		return fmt.Errorf("failed to list Secrets: %w", err)
	}

	var policies secrethorv1alpha1.SecretPolicyList
	if err := c.Client.List(ctx, &policies); err != nil {
		return fmt.Errorf("failed to list SecretPolicies: %w", err)
	}

	now := time.Now()
	for _, secret := range secrets.Items {
		for _, policy := range policies.Items {
			if !namespaceAllowed(secret.Namespace, policy.Spec.AllowedNamespaces) {
				continue
			}

			if policy.Spec.MaxAgeDays == 0 {
				continue
			}

			age := now.Sub(secret.CreationTimestamp.Time).Hours() / 24
			if int(age) > policy.Spec.MaxAgeDays {
				logger.Info("Expired secret detected",
					"name", secret.Name,
					"namespace", secret.Namespace,
					"age", int(age),
					"maxAgeDays", policy.Spec.MaxAgeDays,
					"policy", policy.Name,
				)

				// Add annotation
				if secret.Annotations == nil {
					secret.Annotations = map[string]string{}
				}
				secret.Annotations["secrethor.io/expired"] = "true"
				if err := c.Client.Update(ctx, &secret); err != nil {
					logger.Error(err, "Failed to annotate expired secret")
				}

				// Emit event
				c.Recorder.Event(&secret, corev1.EventTypeWarning, "SecretExpired",
					fmt.Sprintf("Secret %q has expired (age: %d days, max allowed: %d)", secret.Name, int(age), policy.Spec.MaxAgeDays))

				// Set Prometheus metric
				expiredSecrets.WithLabelValues(secret.Namespace, secret.Name).Set(1)
				break
			}
		}
	}

	return nil
}

func namespaceAllowed(ns string, allowed []string) bool {
	for _, n := range allowed {
		if n == ns {
			return true
		}
	}
	return false
}
