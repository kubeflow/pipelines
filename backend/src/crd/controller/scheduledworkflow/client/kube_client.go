// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/record"
)

const (
	successSynced                = "Synced"
	failedSynced                 = "Failed"
	messageResourceSuccessSynced = "Scheduled workflow synced successfull"
	messageResourceFailedSynced  = "Schedulde workflow synced failed"
)

// KubeClient is a client to call the core Kubernetes APIs.
type KubeClient struct {
	// The Kubernetes API client.
	kubeClientSet kubernetes.Interface
	// Recorder is an event recorder for recording Event resources to the Kubernetes API.
	recorder record.EventRecorder
}

// NewKubeClient creates a new client to call the core Kubernetes APIs.
func NewKubeClient(kubeClientSet kubernetes.Interface, recorder record.EventRecorder) *KubeClient {
	return &KubeClient{
		kubeClientSet: kubeClientSet,
		recorder:      recorder,
	}
}

// RecordSyncSuccess records the success of a sync.
func (k *KubeClient) RecordSyncSuccess(swf *swfapi.ScheduledWorkflow, message string) {
	k.recorder.Event(swf, corev1.EventTypeNormal, successSynced,
		fmt.Sprintf("%v: %v", messageResourceSuccessSynced, message))
}

// RecordSyncFailure records the failure of a sync.
func (k *KubeClient) RecordSyncFailure(swf *swfapi.ScheduledWorkflow, message string) {
	k.recorder.Event(swf, corev1.EventTypeWarning, failedSynced,
		fmt.Sprintf("%v: %v", messageResourceFailedSynced, message))
}
