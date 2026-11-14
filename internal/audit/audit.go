// Package audit holds the Secret hygiene rules. It only looks at objects it
// is given and never talks to the cluster.
package audit

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

const (
	ReasonPolicyViolation = "PolicyViolation"
	ReasonExpired         = "Expired"
	ReasonUnused          = "Unused"
)

// Result is the outcome of auditing the Secrets covered by one policy.
type Result struct {
	SecretsAudited int
	Findings       []secrethorv1alpha1.Finding
}

// Covers reports whether the policy applies to Secrets in namespace.
func Covers(policy *secrethorv1alpha1.SecretPolicy, namespace string) bool {
	for _, ns := range policy.Spec.AllowedNamespaces {
		if ns == namespace || ns == secrethorv1alpha1.AllNamespaces {
			return true
		}
	}
	return false
}

var charSets = map[string]string{
	"upper":   "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"lower":   "abcdefghijklmnopqrstuvwxyz",
	"number":  "0123456789",
	"special": "!@#$%^&*()_+-=[]{}|;':\",.<>",
}

// Violations lists the ways in which the content of secret breaks the policy.
// Messages never include Secret values.
func Violations(secret *corev1.Secret, policy *secrethorv1alpha1.SecretPolicy) []string {
	var violations []string
	spec := policy.Spec

	if len(spec.AllowedTypes) > 0 {
		allowed := false
		for _, t := range spec.AllowedTypes {
			if t == secret.Type {
				allowed = true
				break
			}
		}
		if !allowed {
			violations = append(violations, fmt.Sprintf("type %q is not allowed", secret.Type))
		}
	}

	for _, k := range spec.RequiredKeys {
		if _, ok := secret.Data[k]; !ok {
			violations = append(violations, fmt.Sprintf("required key %q is missing", k))
		}
	}

	for _, k := range spec.ForbiddenKeys {
		if _, ok := secret.Data[k]; ok {
			violations = append(violations, fmt.Sprintf("key %q is forbidden", k))
		}
	}

	// Sorted so that the output is stable between audits.
	constrained := make([]string, 0, len(spec.ValueConstraints))
	for k := range spec.ValueConstraints {
		constrained = append(constrained, k)
	}
	sort.Strings(constrained)

	for _, k := range constrained {
		raw, ok := secret.Data[k]
		if !ok {
			continue
		}
		value := string(raw)
		rule := spec.ValueConstraints[k]

		if rule.MinLength != nil && len(value) < *rule.MinLength {
			violations = append(violations, fmt.Sprintf("key %q must have at least %d characters", k, *rule.MinLength))
		}
		for _, class := range rule.MustContain {
			if set, ok := charSets[class]; ok && !strings.ContainsAny(value, set) {
				violations = append(violations, fmt.Sprintf("key %q must contain at least one %s character", k, class))
			}
		}
		if rule.Regex != "" {
			re, err := regexp.Compile(rule.Regex)
			if err != nil {
				violations = append(violations, fmt.Sprintf("invalid regex %q for key %q: %v", rule.Regex, k, err))
			} else if !re.MatchString(value) {
				violations = append(violations, fmt.Sprintf("key %q does not match regex %q", k, rule.Regex))
			}
		}
	}

	return violations
}

// LastRotated is the last time the data of the Secret was written, taken
// from the server-side managed fields. It falls back to the creation time
// when that information is not available.
func LastRotated(secret *corev1.Secret) time.Time {
	var last time.Time
	for _, entry := range secret.ManagedFields {
		if entry.Time == nil || entry.FieldsV1 == nil {
			continue
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(entry.FieldsV1.Raw, &fields); err != nil {
			continue
		}
		_, data := fields["f:data"]
		_, stringData := fields["f:stringData"]
		if (data || stringData) && entry.Time.After(last) {
			last = entry.Time.Time
		}
	}
	if last.IsZero() {
		return secret.CreationTimestamp.Time
	}
	return last
}

// ReferencedSecrets returns the Secrets that some Pod, ServiceAccount or
// Ingress points at.
func ReferencedSecrets(pods []corev1.Pod, serviceAccounts []corev1.ServiceAccount, ingresses []networkingv1.Ingress) map[types.NamespacedName]struct{} {
	referenced := map[types.NamespacedName]struct{}{}
	add := func(namespace, name string) {
		if name != "" {
			referenced[types.NamespacedName{Namespace: namespace, Name: name}] = struct{}{}
		}
	}

	for i := range pods {
		pod := &pods[i]
		ns := pod.Namespace
		for _, ref := range pod.Spec.ImagePullSecrets {
			add(ns, ref.Name)
		}
		for _, vol := range pod.Spec.Volumes {
			if vol.Secret != nil {
				add(ns, vol.Secret.SecretName)
			}
			if vol.Projected != nil {
				for _, src := range vol.Projected.Sources {
					if src.Secret != nil {
						add(ns, src.Secret.Name)
					}
				}
			}
		}
		containerRefs := func(envFrom []corev1.EnvFromSource, env []corev1.EnvVar) {
			for _, src := range envFrom {
				if src.SecretRef != nil {
					add(ns, src.SecretRef.Name)
				}
			}
			for _, e := range env {
				if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
					add(ns, e.ValueFrom.SecretKeyRef.Name)
				}
			}
		}
		for _, c := range pod.Spec.InitContainers {
			containerRefs(c.EnvFrom, c.Env)
		}
		for _, c := range pod.Spec.Containers {
			containerRefs(c.EnvFrom, c.Env)
		}
		for _, c := range pod.Spec.EphemeralContainers {
			containerRefs(c.EnvFrom, c.Env)
		}
	}

	for i := range serviceAccounts {
		sa := &serviceAccounts[i]
		for _, ref := range sa.Secrets {
			add(sa.Namespace, ref.Name)
		}
		for _, ref := range sa.ImagePullSecrets {
			add(sa.Namespace, ref.Name)
		}
	}

	for i := range ingresses {
		ing := &ingresses[i]
		for _, tls := range ing.Spec.TLS {
			add(ing.Namespace, tls.SecretName)
		}
	}

	return referenced
}

// Secret types that are created and consumed by the platform itself, so not
// being referenced by a workload says nothing about them.
var neverReportedUnused = map[corev1.SecretType]struct{}{
	corev1.SecretTypeServiceAccountToken: {},
	corev1.SecretTypeBootstrapToken:      {},
	"helm.sh/release.v1":                 {},
}

// Run audits the Secrets covered by the policy. referenced is only consulted
// when the policy asks for unused detection.
func Run(policy *secrethorv1alpha1.SecretPolicy, secrets []corev1.Secret, referenced map[types.NamespacedName]struct{}, now time.Time) Result {
	var result Result
	maxAge := time.Duration(policy.Spec.MaxAgeDays) * 24 * time.Hour

	for i := range secrets {
		secret := &secrets[i]
		if !Covers(policy, secret.Namespace) {
			continue
		}
		result.SecretsAudited++

		report := func(reason, message string) {
			result.Findings = append(result.Findings, secrethorv1alpha1.Finding{
				Namespace: secret.Namespace,
				Name:      secret.Name,
				Reason:    reason,
				Message:   message,
			})
		}

		if violations := Violations(secret, policy); len(violations) > 0 {
			report(ReasonPolicyViolation, strings.Join(violations, "; "))
		}

		if maxAge > 0 {
			if age := now.Sub(LastRotated(secret)); age > maxAge {
				report(ReasonExpired, fmt.Sprintf("data last written %d days ago, max allowed is %d",
					int(age.Hours()/24), policy.Spec.MaxAgeDays))
			}
		}

		if policy.Spec.DetectUnused {
			_, used := referenced[types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}]
			_, exempt := neverReportedUnused[secret.Type]
			if !used && !exempt {
				report(ReasonUnused, "not referenced by any Pod, ServiceAccount or Ingress")
			}
		}
	}

	return result
}
