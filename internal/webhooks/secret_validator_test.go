package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

// policy builds an enforcing policy; auditPolicy builds a report-only one.
func policy(name string, spec secrethorv1alpha1.SecretPolicySpec) *secrethorv1alpha1.SecretPolicy {
	spec.Enforcement = secrethorv1alpha1.EnforcementEnforce
	return &secrethorv1alpha1.SecretPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       spec,
	}
}

func auditPolicy(name string, spec secrethorv1alpha1.SecretPolicySpec) *secrethorv1alpha1.SecretPolicy {
	return &secrethorv1alpha1.SecretPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       spec,
	}
}

func handle(t *testing.T, secret *corev1.Secret, policies ...client.Object) admission.Response {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := secrethorv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(secret)
	if err != nil {
		t.Fatal(err)
	}

	v := &SecretValidator{
		Client:  fake.NewClientBuilder().WithScheme(scheme).WithObjects(policies...).Build(),
		decoder: admission.NewDecoder(scheme),
	}
	return v.Handle(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: secret.Namespace,
			Object:    runtime.RawExtension{Raw: raw},
		},
	})
}

func secretWith(ns string, data map[string]string) *corev1.Secret {
	s := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: ns},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{},
	}
	for k, v := range data {
		s.Data[k] = []byte(v)
	}
	return s
}

func TestRequiredKeys(t *testing.T) {
	p := policy("p", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
		RequiredKeys:      []string{"username", "password"},
	})

	tests := []struct {
		name        string
		data        map[string]string
		wantAllowed bool
	}{
		{"all required keys present", map[string]string{"username": "u", "password": "p"}, true},
		{"one required key missing", map[string]string{"username": "u"}, false},
		{"no data at all", nil, false},
		{"extra key besides the required ones", map[string]string{"username": "u", "password": "p", "host": "h"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := handle(t, secretWith("prod", tt.data), p)
			if resp.Allowed != tt.wantAllowed {
				t.Errorf("Allowed = %v, want %v (message: %s)", resp.Allowed, tt.wantAllowed, resp.Result.Message)
			}
		})
	}
}

func TestAllPoliciesMatchingNamespaceAreEnforced(t *testing.T) {
	strict := policy("a-strict", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
		ForbiddenKeys:     []string{"token"},
	})
	lax := policy("b-lax", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
	})

	resp := handle(t, secretWith("prod", map[string]string{"token": "x"}), strict, lax)
	if resp.Allowed {
		t.Errorf("secret with a key forbidden by policy %q was allowed", strict.Name)
	}
}

func TestNamespaceNotCoveredIsDeniedWhenSomePolicyEnforces(t *testing.T) {
	p := policy("p", secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}})

	resp := handle(t, secretWith("dev", map[string]string{"k": "v"}), p)
	if resp.Allowed {
		t.Error("secret in namespace not listed by any policy was allowed")
	}
}

func TestNoPoliciesAllowsSecret(t *testing.T) {
	resp := handle(t, secretWith("dev", map[string]string{"k": "v"}))
	if !resp.Allowed {
		t.Errorf("secret denied with no policies: %s", resp.Result.Message)
	}
}

func TestNamespaceNotCoveredIsAllowedWhenAllPoliciesOnlyAudit(t *testing.T) {
	p := auditPolicy("p", secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}})

	resp := handle(t, secretWith("dev", map[string]string{"k": "v"}), p)
	if !resp.Allowed {
		t.Errorf("secret denied although no policy enforces: %s", resp.Result.Message)
	}
}

func TestAuditPolicyAllowsViolatingSecretWithWarning(t *testing.T) {
	p := auditPolicy("p", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
		ForbiddenKeys:     []string{"token"},
	})

	resp := handle(t, secretWith("prod", map[string]string{"token": "x"}), p)
	if !resp.Allowed {
		t.Fatalf("audit-only policy denied the secret: %s", resp.Result.Message)
	}
	if len(resp.Warnings) != 1 {
		t.Errorf("warnings = %v, want one warning about the forbidden key", resp.Warnings)
	}
}

func TestCompliantSecretGetsNoWarnings(t *testing.T) {
	p := auditPolicy("p", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
		ForbiddenKeys:     []string{"token"},
	})

	resp := handle(t, secretWith("prod", map[string]string{"k": "v"}), p)
	if !resp.Allowed || len(resp.Warnings) != 0 {
		t.Errorf("Allowed = %v, warnings = %v, want allowed without warnings", resp.Allowed, resp.Warnings)
	}
}

func TestEnforcingPolicyDeniesEvenWhenAnotherOnlyAudits(t *testing.T) {
	enforcing := policy("a-enforcing", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
		ForbiddenKeys:     []string{"token"},
	})
	auditing := auditPolicy("b-auditing", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
		RequiredKeys:      []string{"owner"},
	})

	resp := handle(t, secretWith("prod", map[string]string{"token": "x"}), enforcing, auditing)
	if resp.Allowed {
		t.Error("secret breaking an enforcing policy was allowed")
	}
}

func TestWildcardPolicyCoversEveryNamespace(t *testing.T) {
	p := policy("p", secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"*"},
		ForbiddenKeys:     []string{"token"},
	})

	resp := handle(t, secretWith("anything", map[string]string{"token": "x"}), p)
	if resp.Allowed {
		t.Error("secret breaking a wildcard enforcing policy was allowed")
	}
}
