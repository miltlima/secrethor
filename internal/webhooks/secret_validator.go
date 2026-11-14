package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	secrethorv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
	"github.com/miltlima/secrethor/internal/audit"
)

// SecretValidator checks Secrets against the SecretPolicies at admission
// time. Only policies in Enforce mode can reject a Secret; policies in Audit
// mode answer with warnings.
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

	var (
		covered     bool
		anyEnforces bool
		denials     []string
		warnings    []string
	)
	for i := range policies.Items {
		policy := &policies.Items[i]
		enforces := policy.Spec.Enforcement == secrethorv1alpha1.EnforcementEnforce
		anyEnforces = anyEnforces || enforces

		if !audit.Covers(policy, targetNS) {
			continue
		}
		covered = true

		for _, violation := range audit.Violations(secret, policy) {
			line := fmt.Sprintf("%s (policy: %q)", violation, policy.Name)
			if enforces {
				denials = append(denials, "- "+line)
			} else {
				warnings = append(warnings, "Secrethor: "+line)
			}
		}
	}

	// With an enforcing policy in place, namespaces that no policy lists
	// are closed to Secrets.
	if !covered && anyEnforces {
		return admission.Denied(fmt.Sprintf(`
Secret rejected by Secrethor policy webhook!

Reason:
- Namespace %q is not allowed by any existing SecretPolicy.
Suggestion:
- Add %q to the .spec.allowedNamespaces field of a SecretPolicy.
`, targetNS, targetNS))
	}

	if len(denials) > 0 {
		return admission.Denied(fmt.Sprintf(`
Secret rejected by Secrethor policy webhook!

The following violations were found:
%s
Suggestion:
- Review the Secret content and update to comply with the policy.
`, strings.Join(denials, "\n"))).WithWarnings(warnings...)
	}

	return admission.Allowed("secret passed all enforced policy checks").WithWarnings(warnings...)
}

func (v *SecretValidator) InjectClient(c client.Client) error {
	v.Client = c
	return nil
}

func (v *SecretValidator) InjectDecoder(d admission.Decoder) error {
	v.decoder = d
	return nil
}
