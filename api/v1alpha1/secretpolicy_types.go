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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnforcementMode says what happens when a Secret breaks a policy at admission time.
// +kubebuilder:validation:Enum=Audit;Enforce
type EnforcementMode string

const (
	// EnforcementAudit never blocks a Secret: problems are only reported in the
	// policy status, in metrics and as admission warnings.
	EnforcementAudit EnforcementMode = "Audit"
	// EnforcementEnforce rejects Secrets that break the policy (requires the webhook).
	EnforcementEnforce EnforcementMode = "Enforce"
)

// AllNamespaces in allowedNamespaces makes the policy cover every namespace.
const AllNamespaces = "*"

// SecretPolicySpec defines the desired state of SecretPolicy
type SecretPolicySpec struct {
	// Enforcement is Audit (default, report only) or Enforce (reject at admission).
	// +kubebuilder:default=Audit
	Enforcement EnforcementMode `json:"enforcement,omitempty"`

	// AllowedNamespaces lists the namespaces covered by the policy. "*" covers all.
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`

	// MaxAgeDays is the maximum number of days since the Secret data was last
	// written before it is reported as expired. 0 disables the check.
	// +kubebuilder:validation:Minimum=0
	MaxAgeDays int `json:"maxAgeDays,omitempty"`

	// DetectUnused reports Secrets that no Pod, ServiceAccount or Ingress references.
	DetectUnused bool `json:"detectUnused,omitempty"`

	AllowedTypes []corev1.SecretType `json:"allowedTypes,omitempty"`
	RequiredKeys []string            `json:"requiredKeys,omitempty"`

	ForbiddenKeys []string `json:"forbiddenKeys,omitempty"`

	ValueConstraints map[string]ValueConstraint `json:"valueConstraints,omitempty"`
}

type ValueConstraint struct {
	MinLength   *int     `json:"minLength,omitempty"`
	MustContain []string `json:"mustContain,omitempty"`
	Regex       string   `json:"regex,omitempty"`
}

// Finding is one hygiene problem found in one Secret.
type Finding struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// Reason is PolicyViolation, Expired or Unused.
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// FindingSummary counts the Secrets affected by each kind of problem.
type FindingSummary struct {
	PolicyViolations int `json:"policyViolations"`
	Expired          int `json:"expired"`
	Unused           int `json:"unused"`
}

// SecretPolicyStatus defines the observed state of SecretPolicy
type SecretPolicyStatus struct {
	// LastAuditTime is when the Secrets covered by the policy were last audited.
	LastAuditTime *metav1.Time `json:"lastAuditTime,omitempty"`
	// SecretsAudited is the number of Secrets covered by the policy in the last audit.
	SecretsAudited int `json:"secretsAudited"`
	// Summary counts every finding of the last audit, including the ones
	// left out of Findings.
	Summary FindingSummary `json:"summary"`
	// Findings lists the problems found, capped to keep the object small.
	Findings []Finding `json:"findings,omitempty"`
	// FindingsTruncated is true when Findings does not hold every finding.
	FindingsTruncated bool `json:"findingsTruncated,omitempty"`
}

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.enforcement`
// +kubebuilder:printcolumn:name="Audited",type=integer,JSONPath=`.status.secretsAudited`
// +kubebuilder:printcolumn:name="Violations",type=integer,JSONPath=`.status.summary.policyViolations`
// +kubebuilder:printcolumn:name="Expired",type=integer,JSONPath=`.status.summary.expired`
// +kubebuilder:printcolumn:name="Unused",type=integer,JSONPath=`.status.summary.unused`
// +kubebuilder:printcolumn:name="Last Audit",type=date,JSONPath=`.status.lastAuditTime`
// +kubebuilder:webhook:path=/validate-v1-secret,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=secrets,verbs=create;update,versions=v1,name=secrets.secrethor.dev,admissionReviewVersions=v1

// SecretPolicy is the Schema for the secretpolicies API
type SecretPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretPolicySpec   `json:"spec,omitempty"`
	Status SecretPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretPolicyList contains a list of SecretPolicy
type SecretPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretPolicy{}, &SecretPolicyList{})
}
