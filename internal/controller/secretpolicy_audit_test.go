package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	secretsv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

func newReconciler(t *testing.T, objs ...client.Object) *SecretPolicyReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := secretsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	return &SecretPolicyReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(&secretsv1alpha1.SecretPolicy{}).
			Build(),
		Scheme:      scheme,
		Interval:    15 * time.Minute,
		MaxFindings: 100,
	}
}

func auditPolicy(name string, spec secretsv1alpha1.SecretPolicySpec) *secretsv1alpha1.SecretPolicy {
	return &secretsv1alpha1.SecretPolicy{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: spec}
}

func agedSecret(ns, name string, days int) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         ns,
			Name:              name,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Duration(days) * 24 * time.Hour)),
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func reconcilePolicy(t *testing.T, r *SecretPolicyReconciler, name string) (ctrl.Result, secretsv1alpha1.SecretPolicyStatus) {
	t.Helper()
	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: name}})
	if err != nil {
		t.Fatal(err)
	}
	var p secretsv1alpha1.SecretPolicy
	if err := r.Get(context.Background(), types.NamespacedName{Name: name}, &p); err != nil {
		t.Fatal(err)
	}
	return res, p.Status
}

func TestReconcileWritesAuditResultToStatus(t *testing.T) {
	leaky := agedSecret("prod", "leaky", 1)
	leaky.Data = map[string][]byte{"token": []byte("x")}
	r := newReconciler(t,
		auditPolicy("status", secretsv1alpha1.SecretPolicySpec{
			AllowedNamespaces: []string{"prod"},
			MaxAgeDays:        30,
			ForbiddenKeys:     []string{"token"},
		}),
		agedSecret("prod", "old", 45),
		agedSecret("prod", "fresh", 1),
		leaky,
		agedSecret("dev", "old-elsewhere", 45),
	)

	res, status := reconcilePolicy(t, r, "status")

	if status.SecretsAudited != 3 {
		t.Errorf("SecretsAudited = %d, want 3", status.SecretsAudited)
	}
	if status.Summary != (secretsv1alpha1.FindingSummary{PolicyViolations: 1, Expired: 1}) {
		t.Errorf("Summary = %+v, want 1 policy violation and 1 expired", status.Summary)
	}
	got := map[string]string{}
	for _, f := range status.Findings {
		got[f.Namespace+"/"+f.Name] = f.Reason
	}
	if len(got) != 2 || got["prod/old"] != "Expired" || got["prod/leaky"] != "PolicyViolation" {
		t.Errorf("Findings = %v, want prod/old Expired and prod/leaky PolicyViolation", got)
	}
	if status.LastAuditTime == nil {
		t.Error("LastAuditTime not set")
	}
	if res.RequeueAfter != 15*time.Minute {
		t.Errorf("RequeueAfter = %v, want the audit interval (15m)", res.RequeueAfter)
	}
}

func TestReconcileNeverWritesToSecrets(t *testing.T) {
	r := newReconciler(t,
		auditPolicy("readonly", secretsv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, MaxAgeDays: 30}),
		agedSecret("prod", "old", 45),
	)
	var before corev1.Secret
	key := types.NamespacedName{Namespace: "prod", Name: "old"}
	if err := r.Get(context.Background(), key, &before); err != nil {
		t.Fatal(err)
	}

	reconcilePolicy(t, r, "readonly")

	var after corev1.Secret
	if err := r.Get(context.Background(), key, &after); err != nil {
		t.Fatal(err)
	}
	if after.ResourceVersion != before.ResourceVersion {
		t.Errorf("secret was modified by the audit: resourceVersion %s -> %s", before.ResourceVersion, after.ResourceVersion)
	}
}

func TestReconcileCapsFindingsButCountsAll(t *testing.T) {
	objs := []client.Object{
		auditPolicy("cap", secretsv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, MaxAgeDays: 30}),
	}
	for i := 0; i < 5; i++ {
		objs = append(objs, agedSecret("prod", fmt.Sprintf("old-%d", i), 45))
	}
	r := newReconciler(t, objs...)
	r.MaxFindings = 3

	_, status := reconcilePolicy(t, r, "cap")

	if len(status.Findings) != 3 {
		t.Errorf("len(Findings) = %d, want 3", len(status.Findings))
	}
	if status.Summary.Expired != 5 {
		t.Errorf("Summary.Expired = %d, want 5", status.Summary.Expired)
	}
	if !status.FindingsTruncated {
		t.Error("FindingsTruncated = false, want true")
	}
}

func TestReconcileSkipsSecretsInUseWhenDetectingUnused(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "app"},
		Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
			Name:         "v",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "mounted"}},
		}}},
	}
	r := newReconciler(t,
		auditPolicy("unused", secretsv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, DetectUnused: true}),
		agedSecret("prod", "mounted", 1),
		agedSecret("prod", "orphan", 1),
		pod,
	)

	_, status := reconcilePolicy(t, r, "unused")

	if status.Summary.Unused != 1 || len(status.Findings) != 1 || status.Findings[0].Name != "orphan" {
		t.Errorf("Summary = %+v, Findings = %v, want only prod/orphan unused", status.Summary, status.Findings)
	}
}

func TestReconcileExposesFindingsAsMetrics(t *testing.T) {
	r := newReconciler(t,
		auditPolicy("metrics", secretsv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, MaxAgeDays: 30}),
		agedSecret("prod", "old-1", 45),
		agedSecret("prod", "old-2", 45),
		agedSecret("prod", "fresh", 1),
	)

	reconcilePolicy(t, r, "metrics")

	if got := testutil.ToFloat64(secretFindings.WithLabelValues("metrics", "prod", "Expired")); got != 2 {
		t.Errorf("secrethor_secret_findings{policy=metrics,namespace=prod,reason=Expired} = %v, want 2", got)
	}
	if got := testutil.ToFloat64(secretsAudited.WithLabelValues("metrics")); got != 3 {
		t.Errorf("secrethor_secrets_audited{policy=metrics} = %v, want 3", got)
	}
}

// seriesForPolicy counts the secrethor series served for one policy.
func seriesForPolicy(t *testing.T, policy string) int {
	t.Helper()
	families, err := metrics.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range families {
		if !strings.HasPrefix(f.GetName(), "secrethor_") {
			continue
		}
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "policy" && l.GetValue() == policy {
					n++
				}
			}
		}
	}
	return n
}

func TestReconcileDropsMetricsOfDeletedPolicy(t *testing.T) {
	p := auditPolicy("gone", secretsv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, MaxAgeDays: 30})
	r := newReconciler(t, p, agedSecret("prod", "old", 45))
	reconcilePolicy(t, r, "gone")
	if got := seriesForPolicy(t, "gone"); got != 2 {
		t.Fatalf("series for the policy after its audit = %d, want 2 (findings and audited)", got)
	}

	if err := r.Delete(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "gone"}}); err != nil {
		t.Fatal(err)
	}

	if got := seriesForPolicy(t, "gone"); got != 0 {
		t.Errorf("series for the deleted policy = %d, want 0", got)
	}
}

func TestReconcileDropsMetricsOfResolvedFindings(t *testing.T) {
	old := agedSecret("prod", "old", 45)
	r := newReconciler(t,
		auditPolicy("resolved", secretsv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, MaxAgeDays: 30}),
		old,
	)
	reconcilePolicy(t, r, "resolved")

	if err := r.Delete(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	reconcilePolicy(t, r, "resolved")

	// Only the audited gauge is left; the Expired series must be gone, not stuck at 1.
	if got := seriesForPolicy(t, "resolved"); got != 1 {
		t.Errorf("series for the policy after the finding was resolved = %d, want 1", got)
	}
}
