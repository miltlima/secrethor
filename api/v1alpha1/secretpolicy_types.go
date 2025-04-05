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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecretPolicySpec defines the desired state of SecretPolicy
type SecretPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	MaxAgeDays        int      `json:"maxAgeDays,omitempty"`
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`

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

// SecretPolicyStatus defines the observed state of SecretPolicy
type SecretPolicyStatus struct {
	Violations []string `json:"violations,omitempty"`
}

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
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
