package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

type SecretValidator struct {
	Client  client.Client
	decoder admission.Decoder
}

func (v *SecretValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	secret := &corev1.Secret{}
	if err := v.decoder.Decode(req, secret); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	var policies secrethorv1alpha1.SecretPolicyList
	if err := v.Client.List(ctx, &policies); err != nil {
		return admission.Denied("cannot list SecretPolicies")
	}

	targetNS := req.Namespace
	if targetNS == "" {
		targetNS = secret.Namespace
	}

	if len(policies.Items) == 0 {
		return admission.Allowed("no SecretPolicy present — allowing secret by default")
	}

	var matched []secrethorv1alpha1.SecretPolicy
	for _, p := range policies.Items {
		for _, ns := range p.Spec.AllowedNamespaces {
			if ns == targetNS {
				matched = append(matched, p)
				break
			}
		}
	}
	if len(matched) == 0 {
		return admission.Denied(fmt.Sprintf(`
		
Secret rejected by Secrethor policy webhook!

Reason:
- Namespace %q is not allowed by any existing SecretPolicy.
Suggestion:
- Add %q to the .spec.allowedNamespaces field of a SecretPolicy.
`, targetNS, targetNS))
	}

	// Every policy that covers the namespace must be satisfied.
	var violations []string
	for _, policy := range matched {
		violations = append(violations, policyViolations(secret, policy)...)
	}

	if len(violations) > 0 {
		return admission.Denied(fmt.Sprintf(`
Secret rejected by Secrethor policy webhook!

The following violations were found:
%s
Suggestion:
- Review the Secret content and update to comply with the policy.
`, strings.Join(violations, "\n")))
	}

	return admission.Allowed("secret passed all policy checks")
}

// policyViolations returns the ways in which secret breaks a single policy.
func policyViolations(secret *corev1.Secret, policy secrethorv1alpha1.SecretPolicy) []string {
	var violations []string

	if len(policy.Spec.AllowedTypes) > 0 {
		allowed := false
		for _, t := range policy.Spec.AllowedTypes {
			if t == secret.Type {
				allowed = true
				break
			}
		}
		if !allowed {
			violations = append(violations, fmt.Sprintf("- Type %q is not allowed (policy: %q)", secret.Type, policy.Name))
		}
	}

	for _, requiredKey := range policy.Spec.RequiredKeys {
		if _, ok := secret.Data[requiredKey]; !ok {
			violations = append(violations, fmt.Sprintf("- Required key %q is missing (policy: %q)", requiredKey, policy.Name))
		}
	}

	for _, forbiddenKey := range policy.Spec.ForbiddenKeys {
		if _, ok := secret.Data[forbiddenKey]; ok {
			violations = append(violations, fmt.Sprintf("- Key %q is forbidden (policy: %q)", forbiddenKey, policy.Name))
		}
	}

	charSets := map[string]string{
		"upper":   "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"lower":   "abcdefghijklmnopqrstuvwxyz",
		"number":  "0123456789",
		"special": "!@#$%^&*()_+-=[]{}|;':\",.<>",
	}

	for key, rule := range policy.Spec.ValueConstraints {
		valueBytes, exists := secret.Data[key]
		if !exists {
			continue
		}
		value := string(valueBytes)

		if rule.MinLength != nil && len(value) < *rule.MinLength {
			violations = append(violations, fmt.Sprintf("- Key %q must have at least %d characters", key, *rule.MinLength))
		}

		for _, check := range rule.MustContain {
			if charSet, ok := charSets[check]; ok && !strings.ContainsAny(value, charSet) {
				violations = append(violations, fmt.Sprintf("- Key %q must contain at least one %s character", key, check))
			}
		}

		if rule.Regex != "" {
			re, err := regexp.Compile(rule.Regex)
			if err != nil {
				violations = append(violations, fmt.Sprintf("- Invalid regex %q for key %q: %v", rule.Regex, key, err))
			} else if !re.MatchString(value) {
				violations = append(violations, fmt.Sprintf("- Key %q does not match regex %q", key, rule.Regex))
			}
		}
	}

	return violations
}

func (v *SecretValidator) InjectClient(c client.Client) error {
	v.Client = c
	return nil
}
func (v *SecretValidator) InjectDecoder(d admission.Decoder) error {
	v.decoder = d
	return nil
}
