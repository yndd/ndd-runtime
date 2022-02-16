// +build !ignore_autogenerated

/*
Copyright 2021 NDD.

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

package v1

import (
	"github.com/karimra/gnmic/target"
	"github.com/karimra/gnmic/types"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Condition.
func (in *Condition) DeepCopy() *Condition {
	if in == nil {
		return nil
	}
	out := new(Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConditionedStatus) DeepCopyInto(out *ConditionedStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConditionedStatus.
func (in *ConditionedStatus) DeepCopy() *ConditionedStatus {
	if in == nil {
		return nil
	}
	out := new(ConditionedStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkNodeStatus) DeepCopyInto(out *NetworkNodeStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkNodeStatus.
func (in *NetworkNodeStatus) DeepCopy() *NetworkNodeStatus {
	if in == nil {
		return nil
	}
	out := new(NetworkNodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkNodeUsage) DeepCopyInto(out *NetworkNodeUsage) {
	*out = *in
	out.NetworkNodeReference = in.NetworkNodeReference
	out.ResourceReference = in.ResourceReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkNodeUsage.
func (in *NetworkNodeUsage) DeepCopy() *NetworkNodeUsage {
	if in == nil {
		return nil
	}
	out := new(NetworkNodeUsage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Reference) DeepCopyInto(out *Reference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Reference.
func (in *Reference) DeepCopy() *Reference {
	if in == nil {
		return nil
	}
	out := new(Reference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Register) DeepCopyInto(out *Register) {
	*out = *in
	if in.Subscriptions != nil {
		in, out := &in.Subscriptions, &out.Subscriptions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExceptionPaths != nil {
		in, out := &in.ExceptionPaths, &out.ExceptionPaths
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExplicitExceptionPaths != nil {
		in, out := &in.ExplicitExceptionPaths, &out.ExplicitExceptionPaths
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Register.
func (in *Register) DeepCopy() *Register {
	if in == nil {
		return nil
	}
	out := new(Register)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceName) DeepCopyInto(out *ResourceName) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceName.
func (in *ResourceName) DeepCopy() *ResourceName {
	if in == nil {
		return nil
	}
	out := new(ResourceName)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceSpec) DeepCopyInto(out *ResourceSpec) {
	*out = *in
	if in.NetworkNodeReference != nil {
		in, out := &in.NetworkNodeReference, &out.NetworkNodeReference
		*out = new(Reference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceSpec.
func (in *ResourceSpec) DeepCopy() *ResourceSpec {
	if in == nil {
		return nil
	}
	out := new(ResourceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceStatus) DeepCopyInto(out *ResourceStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	if in.Target != nil {
		in, out := &in.Target, &out.Target
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExternalLeafRefs != nil {
		in, out := &in.ExternalLeafRefs, &out.ExternalLeafRefs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ResourceIndexes != nil {
		in, out := &in.ResourceIndexes, &out.ResourceIndexes
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceStatus.
func (in *ResourceStatus) DeepCopy() *ResourceStatus {
	if in == nil {
		return nil
	}
	out := new(ResourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Selector) DeepCopyInto(out *Selector) {
	*out = *in
	if in.MatchLabels != nil {
		in, out := &in.MatchLabels, &out.MatchLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.MatchControllerRef != nil {
		in, out := &in.MatchControllerRef, &out.MatchControllerRef
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Selector.
func (in *Selector) DeepCopy() *Selector {
	if in == nil {
		return nil
	}
	out := new(Selector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Target) DeepCopyInto(out *Target) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(types.TargetConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Target != nil {
		in, out := &in.Target, &out.Target
		*out = new(target.Target)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Target.
func (in *Target) DeepCopy() *Target {
	if in == nil {
		return nil
	}
	out := new(Target)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TransactionDeviceStatus) DeepCopyInto(out *TransactionDeviceStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TransactionDeviceStatus.
func (in *TransactionDeviceStatus) DeepCopy() *TransactionDeviceStatus {
	if in == nil {
		return nil
	}
	out := new(TransactionDeviceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TransactionResourceStatus) DeepCopyInto(out *TransactionResourceStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	if in.Device != nil {
		in, out := &in.Device, &out.Device
		*out = make(map[string]*TransactionDeviceStatus, len(*in))
		for key, val := range *in {
			var outVal *TransactionDeviceStatus
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(TransactionDeviceStatus)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TransactionResourceStatus.
func (in *TransactionResourceStatus) DeepCopy() *TransactionResourceStatus {
	if in == nil {
		return nil
	}
	out := new(TransactionResourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TypedReference) DeepCopyInto(out *TypedReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TypedReference.
func (in *TypedReference) DeepCopy() *TypedReference {
	if in == nil {
		return nil
	}
	out := new(TypedReference)
	in.DeepCopyInto(out)
	return out
}
