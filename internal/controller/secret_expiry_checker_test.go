package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

const expiredAnnotation = "secrethor.io/expired"

func newChecker(t *testing.T, objs ...client.Object) (*SecretExpiryChecker, *record.FakeRecorder) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := secrethorv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	recorder := record.NewFakeRecorder(10)
	return &SecretExpiryChecker{
		Client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		Recorder: recorder,
	}, recorder
}

func maxAgePolicy(days int, namespaces ...string) *secrethorv1alpha1.SecretPolicy {
	return &secrethorv1alpha1.SecretPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "max-age"},
		Spec: secrethorv1alpha1.SecretPolicySpec{
			AllowedNamespaces: namespaces,
			MaxAgeDays:        days,
		},
	}
}

func secretAged(name, ns string, days int) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         ns,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Duration(days) * 24 * time.Hour)),
		},
	}
}

func annotations(t *testing.T, c client.Client, ns, name string) map[string]string {
	t.Helper()
	var s corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, &s); err != nil {
		t.Fatal(err)
	}
	return s.Annotations
}

func TestCheckAnnotatesOnlyExpiredSecrets(t *testing.T) {
	checker, _ := newChecker(t,
		maxAgePolicy(30, "prod"),
		secretAged("old", "prod", 45),
		secretAged("fresh", "prod", 5),
		secretAged("old-elsewhere", "dev", 45),
	)

	if err := checker.CheckExpiredSecrets(context.Background()); err != nil {
		t.Fatal(err)
	}

	if got := annotations(t, checker.Client, "prod", "old")[expiredAnnotation]; got != "true" {
		t.Errorf("expired secret annotation = %q, want \"true\"", got)
	}
	if _, ok := annotations(t, checker.Client, "prod", "fresh")[expiredAnnotation]; ok {
		t.Error("secret younger than maxAgeDays was marked expired")
	}
	if _, ok := annotations(t, checker.Client, "dev", "old-elsewhere")[expiredAnnotation]; ok {
		t.Error("secret outside the policy namespaces was marked expired")
	}
}

func TestCheckEmitsExpiryEventOnlyOnce(t *testing.T) {
	checker, recorder := newChecker(t, maxAgePolicy(30, "prod"), secretAged("old", "prod", 45))

	for i := 0; i < 2; i++ {
		if err := checker.CheckExpiredSecrets(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if got := len(recorder.Events); got != 1 {
		t.Errorf("events after two checks = %d, want 1", got)
	}
}

func TestExpiredSecretsMetricIsServedByManagerRegistry(t *testing.T) {
	checker, _ := newChecker(t, maxAgePolicy(30, "prod"), secretAged("old", "prod", 45))

	if err := checker.CheckExpiredSecrets(context.Background()); err != nil {
		t.Fatal(err)
	}

	families, err := metrics.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() == "secrethor_expired_secrets" {
			return
		}
	}
	t.Error("secrethor_expired_secrets not found in the controller-runtime metrics registry")
}

func TestStartRunsACheckImmediately(t *testing.T) {
	checker, _ := newChecker(t, maxAgePolicy(30, "prod"), secretAged("old", "prod", 45))
	checker.Interval = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- checker.Start(ctx) }()

	deadline := time.Now().Add(5 * time.Second)
	for annotations(t, checker.Client, "prod", "old")[expiredAnnotation] != "true" {
		if time.Now().After(deadline) {
			t.Fatal("expired secret not annotated within 5s of Start")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned %v after cancellation, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}
