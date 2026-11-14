/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	secretsv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
	"github.com/miltlima/secrethor/internal/audit"
)

// SecretPolicyReconciler reconciles a SecretPolicy object
type SecretPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// Interval is how often each policy is re-audited.
	Interval time.Duration
	// MaxFindings caps the findings kept in the status of a policy.
	MaxFindings int
}

var (
	secretFindings = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secrethor_secret_findings",
		Help: "Secrets with a hygiene problem, by policy, namespace and reason (PolicyViolation, Expired, Unused)",
	}, []string{"policy", "namespace", "reason"})
	secretsAudited = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "secrethor_secrets_audited",
		Help: "Secrets covered by the policy in its last audit",
	}, []string{"policy"})
)

func init() {
	// The manager only serves the controller-runtime registry.
	metrics.Registry.MustRegister(secretFindings, secretsAudited)
}

const (
	defaultInterval    = time.Hour
	defaultMaxFindings = 100
)

// +kubebuilder:rbac:groups=secrets.secrethor.dev,resources=secretpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=secrets.secrethor.dev,resources=secretpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;pods;serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

// Reconcile audits the Secrets covered by one SecretPolicy and records what
// it finds in the policy status and in metrics. It never writes to Secrets.
func (r *SecretPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var policy secretsv1alpha1.SecretPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if apierrors.IsNotFound(err) {
			dropPolicyMetrics(req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var secrets corev1.SecretList
	if err := r.List(ctx, &secrets); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing secrets: %w", err)
	}

	var referenced map[types.NamespacedName]struct{}
	if policy.Spec.DetectUnused {
		var err error
		if referenced, err = r.referencedSecrets(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}

	now := time.Now()
	result := audit.Run(&policy, secrets.Items, referenced, now)

	// Stable order, so that the status only changes when the findings do.
	sort.Slice(result.Findings, func(i, j int) bool {
		a, b := result.Findings[i], result.Findings[j]
		if a.Namespace != b.Namespace {
			return a.Namespace < b.Namespace
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Reason < b.Reason
	})

	var summary secretsv1alpha1.FindingSummary
	perSeries := map[[2]string]float64{}
	for _, f := range result.Findings {
		switch f.Reason {
		case audit.ReasonPolicyViolation:
			summary.PolicyViolations++
		case audit.ReasonExpired:
			summary.Expired++
		case audit.ReasonUnused:
			summary.Unused++
		}
		perSeries[[2]string{f.Namespace, f.Reason}]++
	}

	dropPolicyMetrics(policy.Name)
	secretsAudited.WithLabelValues(policy.Name).Set(float64(result.SecretsAudited))
	for series, count := range perSeries {
		secretFindings.WithLabelValues(policy.Name, series[0], series[1]).Set(count)
	}

	maxFindings := r.MaxFindings
	if maxFindings <= 0 {
		maxFindings = defaultMaxFindings
	}
	findings := result.Findings
	truncated := len(findings) > maxFindings
	if truncated {
		findings = findings[:maxFindings]
	}

	auditTime := metav1.NewTime(now)
	policy.Status = secretsv1alpha1.SecretPolicyStatus{
		LastAuditTime:     &auditTime,
		SecretsAudited:    result.SecretsAudited,
		Summary:           summary,
		Findings:          findings,
		FindingsTruncated: truncated,
	}
	if err := r.Status().Update(ctx, &policy); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("updating status: %w", err)
	}

	log.Info("Audited secrets",
		"audited", result.SecretsAudited,
		"policyViolations", summary.PolicyViolations,
		"expired", summary.Expired,
		"unused", summary.Unused,
	)

	interval := r.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	return ctrl.Result{RequeueAfter: interval}, nil
}

func (r *SecretPolicyReconciler) referencedSecrets(ctx context.Context) (map[types.NamespacedName]struct{}, error) {
	var pods corev1.PodList
	if err := r.List(ctx, &pods); err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}
	var serviceAccounts corev1.ServiceAccountList
	if err := r.List(ctx, &serviceAccounts); err != nil {
		return nil, fmt.Errorf("listing service accounts: %w", err)
	}
	var ingresses networkingv1.IngressList
	if err := r.List(ctx, &ingresses); err != nil {
		return nil, fmt.Errorf("listing ingresses: %w", err)
	}
	return audit.ReferencedSecrets(pods.Items, serviceAccounts.Items, ingresses.Items), nil
}

func dropPolicyMetrics(policy string) {
	labels := prometheus.Labels{"policy": policy}
	secretFindings.DeletePartialMatch(labels)
	secretsAudited.DeletePartialMatch(labels)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Status updates must not trigger a new audit.
		For(&secretsv1alpha1.SecretPolicy{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
