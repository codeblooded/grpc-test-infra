// +build !ignore_autogenerated

/*
Copyright 2020 gRPC authors.

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
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Build) DeepCopyInto(out *Build) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(string)
		**out = **in
	}
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Build.
func (in *Build) DeepCopy() *Build {
	if in == nil {
		return nil
	}
	out := new(Build)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Client) DeepCopyInto(out *Client) {
	*out = *in
	in.Component.DeepCopyInto(&out.Component)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Client.
func (in *Client) DeepCopy() *Client {
	if in == nil {
		return nil
	}
	out := new(Client)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Clone) DeepCopyInto(out *Clone) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(string)
		**out = **in
	}
	if in.Repo != nil {
		in, out := &in.Repo, &out.Repo
		*out = new(string)
		**out = **in
	}
	if in.GitRef != nil {
		in, out := &in.GitRef, &out.GitRef
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Clone.
func (in *Clone) DeepCopy() *Clone {
	if in == nil {
		return nil
	}
	out := new(Clone)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Component) DeepCopyInto(out *Component) {
	*out = *in
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.Pool != nil {
		in, out := &in.Pool, &out.Pool
		*out = new(string)
		**out = **in
	}
	if in.Clone != nil {
		in, out := &in.Clone, &out.Clone
		*out = new(Clone)
		(*in).DeepCopyInto(*out)
	}
	if in.Build != nil {
		in, out := &in.Build, &out.Build
		*out = new(Build)
		(*in).DeepCopyInto(*out)
	}
	in.Run.DeepCopyInto(&out.Run)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Component.
func (in *Component) DeepCopy() *Component {
	if in == nil {
		return nil
	}
	out := new(Component)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Driver) DeepCopyInto(out *Driver) {
	*out = *in
	in.Component.DeepCopyInto(&out.Component)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Driver.
func (in *Driver) DeepCopy() *Driver {
	if in == nil {
		return nil
	}
	out := new(Driver)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadTest) DeepCopyInto(out *LoadTest) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadTest.
func (in *LoadTest) DeepCopy() *LoadTest {
	if in == nil {
		return nil
	}
	out := new(LoadTest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LoadTest) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadTestList) DeepCopyInto(out *LoadTestList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LoadTest, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadTestList.
func (in *LoadTestList) DeepCopy() *LoadTestList {
	if in == nil {
		return nil
	}
	out := new(LoadTestList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LoadTestList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadTestMissing) DeepCopyInto(out *LoadTestMissing) {
	*out = *in
	if in.Driver != nil {
		in, out := &in.Driver, &out.Driver
		*out = new(Driver)
		(*in).DeepCopyInto(*out)
	}
	if in.Servers != nil {
		in, out := &in.Servers, &out.Servers
		*out = make([]Server, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Clients != nil {
		in, out := &in.Clients, &out.Clients
		*out = make([]Client, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadTestMissing.
func (in *LoadTestMissing) DeepCopy() *LoadTestMissing {
	if in == nil {
		return nil
	}
	out := new(LoadTestMissing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadTestSpec) DeepCopyInto(out *LoadTestSpec) {
	*out = *in
	if in.Driver != nil {
		in, out := &in.Driver, &out.Driver
		*out = new(Driver)
		(*in).DeepCopyInto(*out)
	}
	if in.Servers != nil {
		in, out := &in.Servers, &out.Servers
		*out = make([]Server, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Clients != nil {
		in, out := &in.Clients, &out.Clients
		*out = make([]Client, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Results != nil {
		in, out := &in.Results, &out.Results
		*out = new(Results)
		(*in).DeepCopyInto(*out)
	}
	if in.Scenarios != nil {
		in, out := &in.Scenarios, &out.Scenarios
		*out = make([]Scenario, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadTestSpec.
func (in *LoadTestSpec) DeepCopy() *LoadTestSpec {
	if in == nil {
		return nil
	}
	out := new(LoadTestSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadTestStatus) DeepCopyInto(out *LoadTestStatus) {
	*out = *in
	if in.AcknowledgeTime != nil {
		in, out := &in.AcknowledgeTime, &out.AcknowledgeTime
		*out = (*in).DeepCopy()
	}
	if in.ProvisionTime != nil {
		in, out := &in.ProvisionTime, &out.ProvisionTime
		*out = (*in).DeepCopy()
	}
	if in.PendingTime != nil {
		in, out := &in.PendingTime, &out.PendingTime
		*out = (*in).DeepCopy()
	}
	if in.TerminateTime != nil {
		in, out := &in.TerminateTime, &out.TerminateTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadTestStatus.
func (in *LoadTestStatus) DeepCopy() *LoadTestStatus {
	if in == nil {
		return nil
	}
	out := new(LoadTestStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Results) DeepCopyInto(out *Results) {
	*out = *in
	if in.BigQueryTable != nil {
		in, out := &in.BigQueryTable, &out.BigQueryTable
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Results.
func (in *Results) DeepCopy() *Results {
	if in == nil {
		return nil
	}
	out := new(Results)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Run) DeepCopyInto(out *Run) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(string)
		**out = **in
	}
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]corev1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Run.
func (in *Run) DeepCopy() *Run {
	if in == nil {
		return nil
	}
	out := new(Run)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Scenario) DeepCopyInto(out *Scenario) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Scenario.
func (in *Scenario) DeepCopy() *Scenario {
	if in == nil {
		return nil
	}
	out := new(Scenario)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Server) DeepCopyInto(out *Server) {
	*out = *in
	in.Component.DeepCopyInto(&out.Component)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Server.
func (in *Server) DeepCopy() *Server {
	if in == nil {
		return nil
	}
	out := new(Server)
	in.DeepCopyInto(out)
	return out
}
