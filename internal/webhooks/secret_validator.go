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

	var matched bool
	var policy secrethorv1alpha1.SecretPolicy

	for _, p := range policies.Items {
		for _, ns := range p.Spec.AllowedNamespaces {
			if ns == targetNS {
				matched = true
				policy = p
				break
			}
		}
	}
	if !matched {
		return admission.Denied(fmt.Sprintf(
			`Secret creation denied: namespace %q is not allowed by any SecretPolicy.
		Hint: Add %q to .spec.allowedNamespaces in a SecretPolicy resource.`,
			targetNS, targetNS,
		))
	}

	if len(policy.Spec.AllowedTypes) > 0 {
		allowed := false
		for _, t := range policy.Spec.AllowedTypes {
			if t == secret.Type {
				allowed = true
				break
			}
		}
		if !allowed {
			return admission.Denied(fmt.Sprintf(
				`Secret creation denied: type %q is not allowed by SecretPolicy %q.`,
				secret.Type, policy.Name,
			))
		}
	}

	for _, requiredKey := range policy.Spec.RequiredKeys {
		if _, ok := secret.Data[requiredKey]; !ok {
			return admission.Denied(fmt.Sprintf(`Secret creation denied: key %q is required by SecretPolicy %q.`,
				requiredKey, policy.Name,
			))
		}
	}

	for _, forbiddenKey := range policy.Spec.ForbiddenKeys {
		if _, ok := secret.Data[forbiddenKey]; ok {
			return admission.Denied(fmt.Sprintf(`Secret creation denied: key %q is forbidden by SecretPolicy %q.`,
				forbiddenKey, policy.Name,
			))
		}
	}

	for key, rule := range policy.Spec.ValueConstraints {
		valueBytes, exists := secret.Data[key]
		if !exists {
			continue
		}
		value := string(valueBytes)

		if rule.MinLength != nil && len(value) < *rule.MinLength {
			return admission.Denied(fmt.Sprintf(
				`Secret creation denied: key %q must have at least %d characters according to SecretPolicy %q.`,
				key, *rule.MinLength, policy.Name,
			))
		}

		for _, check := range rule.MustContain {
			switch check {
			case "upper":
				if !strings.ContainsAny(value, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
					return admission.Denied(fmt.Sprintf(
						`Secret creation denied: key %q must contain at least one uppercase letter according to SecretPolicy %q.`,
						key, policy.Name,
					))
				}
			case "lower":
				if !strings.ContainsAny(value, "abcdefghijklmnopqrstuvwxyz") {
					return admission.Denied(fmt.Sprintf(
						`Secret creation denied: key %q must contain at least one lowercase letter according to SecretPolicy %q.`,
						key, policy.Name,
					))
				}
			case "number":
				if !strings.ContainsAny(value, "0123456789") {
					return admission.Denied(fmt.Sprintf(
						`Secret creation denied: key %q must contain at least one number according to SecretPolicy %q.`,
						key, policy.Name,
					))
				}
			case "special":
				if !strings.ContainsAny(value, "!@#$%^&*()_+-=[]{}|;':\",.<>?") {
					return admission.Denied(fmt.Sprintf(
						`Secret creation denied: key %q must contain at least one special character according to SecretPolicy %q.`,
						key, policy.Name,
					))
				}
			}
		}

		if rule.Regex != "" {
			re, err := regexp.Compile(rule.Regex)
			if err != nil {
				return admission.Denied(fmt.Sprintf(`invalid regex %q for key %q in SecretPolicy %q: %v`,
					rule.Regex, key, policy.Name, err,
				))
			}
			if !re.MatchString(value) {
				return admission.Denied(fmt.Sprintf(
					`Secret creation denied: key %q does not match regex %q according to SecretPolicy %q.`,
					key, rule.Regex, policy.Name,
				))
			}
		}
	}
	return admission.Allowed("secret passed all policy checks")
}

func (v *SecretValidator) InjectClient(c client.Client) error {
	v.Client = c
	return nil
}
func (v *SecretValidator) InjectDecoder(d admission.Decoder) error {
	v.decoder = d
	return nil
}
