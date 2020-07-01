// Copyright 2020 gRPC authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orch

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type podCreator interface {
	Create(context.Context, *corev1.Pod, metav1.CreateOptions) (*corev1.Pod, error)
}

type podDeleter interface {
	DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error
}

type podLogGetter interface {
	GetLogs(podName string, opts *corev1.PodLogOptions) *rest.Request
}

type podCreateDeleter interface {
	podCreator
	podDeleter
	podLogGetter
}

type podWatcher interface {
	Watch(context.Context, metav1.ListOptions) (watch.Interface, error)
}
