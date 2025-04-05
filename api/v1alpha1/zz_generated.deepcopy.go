//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretPolicy) DeepCopyInto(out *SecretPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretPolicy.
func (in *SecretPolicy) DeepCopy() *SecretPolicy {
	if in == nil {
		return nil
	}
	out := new(SecretPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SecretPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretPolicyList) DeepCopyInto(out *SecretPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SecretPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretPolicyList.
func (in *SecretPolicyList) DeepCopy() *SecretPolicyList {
	if in == nil {
		return nil
	}
	out := new(SecretPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SecretPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretPolicySpec) DeepCopyInto(out *SecretPolicySpec) {
	*out = *in
	if in.AllowedNamespaces != nil {
		in, out := &in.AllowedNamespaces, &out.AllowedNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AllowedTypes != nil {
		in, out := &in.AllowedTypes, &out.AllowedTypes
		*out = make([]v1.SecretType, len(*in))
		copy(*out, *in)
	}
	if in.RequiredKeys != nil {
		in, out := &in.RequiredKeys, &out.RequiredKeys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ForbiddenKeys != nil {
		in, out := &in.ForbiddenKeys, &out.ForbiddenKeys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ValueConstraints != nil {
		in, out := &in.ValueConstraints, &out.ValueConstraints
		*out = make(map[string]ValueConstraint, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretPolicySpec.
func (in *SecretPolicySpec) DeepCopy() *SecretPolicySpec {
	if in == nil {
		return nil
	}
	out := new(SecretPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretPolicyStatus) DeepCopyInto(out *SecretPolicyStatus) {
	*out = *in
	if in.Violations != nil {
		in, out := &in.Violations, &out.Violations
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretPolicyStatus.
func (in *SecretPolicyStatus) DeepCopy() *SecretPolicyStatus {
	if in == nil {
		return nil
	}
	out := new(SecretPolicyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValueConstraint) DeepCopyInto(out *ValueConstraint) {
	*out = *in
	if in.MinLength != nil {
		in, out := &in.MinLength, &out.MinLength
		*out = new(int)
		**out = **in
	}
	if in.MustContain != nil {
		in, out := &in.MustContain, &out.MustContain
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValueConstraint.
func (in *ValueConstraint) DeepCopy() *ValueConstraint {
	if in == nil {
		return nil
	}
	out := new(ValueConstraint)
	in.DeepCopyInto(out)
	return out
}
