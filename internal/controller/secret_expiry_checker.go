package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

var (
	expiredSecrets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "secrethor_expired_secrets",
			Help: "Secrets older than the maxAgeDays of a SecretPolicy covering their namespace",
		},
		[]string{"namespace", "name"},
	)
)

func init() {
	// The manager only serves the controller-runtime registry.
	metrics.Registry.MustRegister(expiredSecrets)
}

type SecretExpiryChecker struct {
	Client   client.Client
	Recorder record.EventRecorder
	Interval time.Duration
}

const (
	expiredAnnotationKey = "secrethor.io/expired"
	defaultCheckInterval = time.Hour
)

// Start checks for expired secrets immediately and then once per Interval,
// until ctx is cancelled. It lets the manager run the checker as a Runnable,
// so it only runs on the elected leader.
func (c *SecretExpiryChecker) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	interval := c.Interval
	if interval <= 0 {
		interval = defaultCheckInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := c.CheckExpiredSecrets(ctx); err != nil {
			logger.Error(err, "failed to check expired secrets")
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}
	}
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

	// Drop series of secrets that were deleted or rotated since the last check.
	expiredSecrets.Reset()

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

				// Annotate and emit the event only the first time the
				// secret is seen expired.
				if secret.Annotations[expiredAnnotationKey] != "true" {
					original := secret.DeepCopy()
					if secret.Annotations == nil {
						secret.Annotations = map[string]string{}
					}
					secret.Annotations[expiredAnnotationKey] = "true"
					if err := c.Client.Patch(ctx, &secret, client.MergeFrom(original)); err != nil {
						logger.Error(err, "Failed to annotate expired secret",
							"name", secret.Name, "namespace", secret.Namespace)
					} else {
						c.Recorder.Event(&secret, corev1.EventTypeWarning, "SecretExpired",
							fmt.Sprintf("Secret %q has expired (age: %d days, max allowed: %d)", secret.Name, int(age), policy.Spec.MaxAgeDays))
					}
				}

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
