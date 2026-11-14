package audit

import (
	"sort"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

var now = time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)

func daysAgo(d int) metav1.Time {
	return metav1.NewTime(now.Add(-time.Duration(d) * 24 * time.Hour))
}

func policyWith(spec secrethorv1alpha1.SecretPolicySpec) *secrethorv1alpha1.SecretPolicy {
	return &secrethorv1alpha1.SecretPolicy{ObjectMeta: metav1.ObjectMeta{Name: "p"}, Spec: spec}
}

func secret(ns, name string, createdDaysAgo int) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, CreationTimestamp: daysAgo(createdDaysAgo)},
		Type:       corev1.SecretTypeOpaque,
	}
}

func key(ns, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: ns, Name: name}
}

// reasonsBySecret flattens findings to "ns/name" -> sorted reasons.
func reasonsBySecret(r Result) map[string][]string {
	out := map[string][]string{}
	for _, f := range r.Findings {
		k := f.Namespace + "/" + f.Name
		out[k] = append(out[k], f.Reason)
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}

func TestCovers(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		ns         string
		want       bool
	}{
		{"listed namespace", []string{"prod", "staging"}, "staging", true},
		{"unlisted namespace", []string{"prod"}, "dev", false},
		{"wildcard covers any namespace", []string{"*"}, "dev", true},
		{"no namespaces covers nothing", nil, "prod", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := policyWith(secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: tt.namespaces})
			if got := Covers(p, tt.ns); got != tt.want {
				t.Errorf("Covers = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestViolations(t *testing.T) {
	minLen := 8
	p := policyWith(secrethorv1alpha1.SecretPolicySpec{
		AllowedTypes:  []corev1.SecretType{corev1.SecretTypeOpaque},
		RequiredKeys:  []string{"username"},
		ForbiddenKeys: []string{"token"},
		ValueConstraints: map[string]secrethorv1alpha1.ValueConstraint{
			"password": {MinLength: &minLen, MustContain: []string{"number"}, Regex: "^[a-z0-9]+$"},
		},
	})

	tests := []struct {
		name string
		typ  corev1.SecretType
		data map[string]string
		want int
	}{
		{"compliant secret", corev1.SecretTypeOpaque, map[string]string{"username": "u", "password": "abcdefg1"}, 0},
		{"type not allowed", corev1.SecretTypeBasicAuth, map[string]string{"username": "u"}, 1},
		{"required key missing", corev1.SecretTypeOpaque, map[string]string{"other": "x"}, 1},
		{"forbidden key present", corev1.SecretTypeOpaque, map[string]string{"username": "u", "token": "x"}, 1},
		{"value too short", corev1.SecretTypeOpaque, map[string]string{"username": "u", "password": "abc1"}, 1},
		{"value lacks a number", corev1.SecretTypeOpaque, map[string]string{"username": "u", "password": "abcdefgh"}, 1},
		{"value breaks regex", corev1.SecretTypeOpaque, map[string]string{"username": "u", "password": "ABCDEFG1"}, 1},
		{"several problems at once", corev1.SecretTypeBasicAuth, map[string]string{"token": "x"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &corev1.Secret{Type: tt.typ, Data: map[string][]byte{}}
			for k, v := range tt.data {
				s.Data[k] = []byte(v)
			}
			got := Violations(s, p)
			if len(got) != tt.want {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.want)
			}
		})
	}
}

func TestLastRotated(t *testing.T) {
	dataWrite := metav1.ManagedFieldsEntry{
		Time:     ptr(daysAgo(10)),
		FieldsV1: &metav1.FieldsV1{Raw: []byte(`{"f:data":{"f:password":{}}}`)},
	}
	labelWrite := metav1.ManagedFieldsEntry{
		Time:     ptr(daysAgo(2)),
		FieldsV1: &metav1.FieldsV1{Raw: []byte(`{"f:metadata":{"f:labels":{"f:team":{}}}}`)},
	}

	t.Run("uses the last write to the data", func(t *testing.T) {
		s := secret("prod", "s", 90)
		s.ManagedFields = []metav1.ManagedFieldsEntry{dataWrite}
		if got := LastRotated(&s); !got.Equal(daysAgo(10).Time) {
			t.Errorf("LastRotated = %v, want %v", got, daysAgo(10).Time)
		}
	})
	t.Run("ignores writes that did not touch the data", func(t *testing.T) {
		s := secret("prod", "s", 90)
		s.ManagedFields = []metav1.ManagedFieldsEntry{dataWrite, labelWrite}
		if got := LastRotated(&s); !got.Equal(daysAgo(10).Time) {
			t.Errorf("LastRotated = %v, want %v", got, daysAgo(10).Time)
		}
	})
	t.Run("falls back to creation time", func(t *testing.T) {
		s := secret("prod", "s", 90)
		if got := LastRotated(&s); !got.Equal(daysAgo(90).Time) {
			t.Errorf("LastRotated = %v, want %v", got, daysAgo(90).Time)
		}
	})
}

func ptr[T any](v T) *T { return &v }

func TestReferencedSecrets(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "app"},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "pull"}},
			Volumes: []corev1.Volume{
				{VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "vol"}}},
				{VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{{Secret: &corev1.SecretProjection{
						LocalObjectReference: corev1.LocalObjectReference{Name: "projected"},
					}}},
				}}},
			},
			InitContainers: []corev1.Container{{
				EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "init-envfrom"},
				}}},
			}},
			Containers: []corev1.Container{{
				Env: []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "env"},
				}}}},
			}},
		},
	}
	sa := corev1.ServiceAccount{
		ObjectMeta:       metav1.ObjectMeta{Namespace: "prod", Name: "sa"},
		Secrets:          []corev1.ObjectReference{{Name: "sa-secret"}},
		ImagePullSecrets: []corev1.LocalObjectReference{{Name: "sa-pull"}},
	}
	ing := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "web", Name: "ing"},
		Spec:       networkingv1.IngressSpec{TLS: []networkingv1.IngressTLS{{SecretName: "tls"}}},
	}

	got := ReferencedSecrets([]corev1.Pod{pod}, []corev1.ServiceAccount{sa}, []networkingv1.Ingress{ing})

	want := []types.NamespacedName{
		key("prod", "pull"), key("prod", "vol"), key("prod", "projected"),
		key("prod", "init-envfrom"), key("prod", "env"),
		key("prod", "sa-secret"), key("prod", "sa-pull"),
		key("web", "tls"),
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("%s not reported as referenced", k)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d referenced secrets, want %d: %v", len(got), len(want), got)
	}
}

func TestRunOnlyAuditsCoveredNamespaces(t *testing.T) {
	p := policyWith(secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, MaxAgeDays: 30})

	r := Run(p, []corev1.Secret{secret("prod", "a", 1), secret("prod", "b", 90), secret("dev", "c", 90)}, nil, now)

	if r.SecretsAudited != 2 {
		t.Errorf("SecretsAudited = %d, want 2", r.SecretsAudited)
	}
	got := reasonsBySecret(r)
	if len(got) != 1 || len(got["prod/b"]) != 1 || got["prod/b"][0] != ReasonExpired {
		t.Errorf("findings = %v, want only prod/b Expired", got)
	}
}

func TestRunExpiryBoundary(t *testing.T) {
	p := policyWith(secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, MaxAgeDays: 30})

	r := Run(p, []corev1.Secret{secret("prod", "at-limit", 30), secret("prod", "over-limit", 31)}, nil, now)

	got := reasonsBySecret(r)
	if _, ok := got["prod/at-limit"]; ok {
		t.Error("secret exactly maxAgeDays old was reported as expired")
	}
	if _, ok := got["prod/over-limit"]; !ok {
		t.Error("secret older than maxAgeDays was not reported as expired")
	}
}

func TestRunWithoutMaxAgeNeverExpires(t *testing.T) {
	p := policyWith(secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}})

	r := Run(p, []corev1.Secret{secret("prod", "ancient", 3000)}, nil, now)

	if len(r.Findings) != 0 {
		t.Errorf("findings = %v, want none", r.Findings)
	}
}

func TestRunReportsPolicyViolations(t *testing.T) {
	p := policyWith(secrethorv1alpha1.SecretPolicySpec{
		AllowedNamespaces: []string{"prod"},
		ForbiddenKeys:     []string{"token"},
	})
	bad := secret("prod", "bad", 1)
	bad.Data = map[string][]byte{"token": []byte("x")}

	r := Run(p, []corev1.Secret{bad, secret("prod", "good", 1)}, nil, now)

	got := reasonsBySecret(r)
	if len(got) != 1 || len(got["prod/bad"]) != 1 || got["prod/bad"][0] != ReasonPolicyViolation {
		t.Errorf("findings = %v, want only prod/bad PolicyViolation", got)
	}
}

func TestRunUnusedDetection(t *testing.T) {
	referenced := map[types.NamespacedName]struct{}{key("prod", "used"): {}}
	saToken := secret("prod", "sa-token", 1)
	saToken.Type = corev1.SecretTypeServiceAccountToken
	helmRelease := secret("prod", "sh.helm.release.v1.app.v1", 1)
	helmRelease.Type = "helm.sh/release.v1"
	secrets := []corev1.Secret{secret("prod", "used", 1), secret("prod", "orphan", 1), saToken, helmRelease}

	t.Run("reports secrets nothing references", func(t *testing.T) {
		p := policyWith(secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}, DetectUnused: true})
		got := reasonsBySecret(Run(p, secrets, referenced, now))
		if len(got) != 1 || len(got["prod/orphan"]) != 1 || got["prod/orphan"][0] != ReasonUnused {
			t.Errorf("findings = %v, want only prod/orphan Unused", got)
		}
	})
	t.Run("is off unless the policy asks for it", func(t *testing.T) {
		p := policyWith(secrethorv1alpha1.SecretPolicySpec{AllowedNamespaces: []string{"prod"}})
		if got := Run(p, secrets, referenced, now).Findings; len(got) != 0 {
			t.Errorf("findings = %v, want none", got)
		}
	})
}
