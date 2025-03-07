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

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrladmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var scheme *runtime.Scheme

func init() {
	scheme = runtime.NewScheme()
	err := k8sapi.AddToScheme(scheme)
	if err != nil {
		// Panic is okay here because it means there's a code issue and so the package shouldn't initialize.
		panic(fmt.Sprintf("Failed to initialize the Kubernetes API scheme: %v", err))
	}
}

type PipelineVersionsWebhook struct {
	Client ctrlclient.Client
}

var _ ctrladmission.CustomValidator = &PipelineVersionsWebhook{}

func newBadRequestError(msg string) *apierrors.StatusError {
	return &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Code:    http.StatusBadRequest,
			Reason:  metav1.StatusReasonBadRequest,
			Message: msg,
		},
	}
}

func (p *PipelineVersionsWebhook) ValidateCreate(
	ctx context.Context, obj runtime.Object,
) (warnings ctrladmission.Warnings, err error) {
	pipelineVersion, ok := obj.(*k8sapi.PipelineVersion)
	if !ok {
		return nil, newBadRequestError(fmt.Sprintf("Expected a PipelineVersion object but got %T", pipelineVersion))
	}

	pipeline := &k8sapi.Pipeline{}

	err = p.Client.Get(
		ctx, types.NamespacedName{Namespace: pipelineVersion.Namespace, Name: pipelineVersion.Spec.PipelineName}, pipeline,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newBadRequestError("The spec.pipelineName doesn't map to an existing Pipeline object")
		}

		return nil, err
	}

	pipelineSpec, err := json.Marshal(pipelineVersion.Spec.PipelineSpec.Value)
	if err != nil {
		return nil, newBadRequestError(fmt.Sprintf("The pipeline spec is invalid JSON: %v", err))
	}

	tmpl, err := template.NewV2SpecTemplate(pipelineSpec)
	if err != nil {
		return nil, newBadRequestError(fmt.Sprintf("The pipeline spec is invalid: %v", err))
	}

	err = common.ValidatePipelineName(tmpl.V2PipelineName())
	if err != nil {
		return nil, newBadRequestError(err.Error())
	}

	if pipelineVersion.Name != tmpl.V2PipelineName() {
		return nil, newBadRequestError("The object name must match spec.pipelineSpec.pipelineInformation.name")
	}

	return nil, nil
}

func (p *PipelineVersionsWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (ctrladmission.Warnings, error) {
	oldPipelineVersion, ok := oldObj.(*k8sapi.PipelineVersion)
	if !ok {
		return nil, newBadRequestError(fmt.Sprintf("Expected a PipelineVersion but got %T", oldObj))
	}

	newPipelineVersion, ok := newObj.(*k8sapi.PipelineVersion)
	if !ok {
		return nil, newBadRequestError(fmt.Sprintf("Expected a PipelineVersion but got %T", newObj))
	}

	if !reflect.DeepEqual(oldPipelineVersion.Spec, newPipelineVersion.Spec) {
		return nil, newBadRequestError("Pipeline spec is immutable; only metadata changes (labels/annotations) are allowed")
	}

	return nil, nil
}

func (p *PipelineVersionsWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (ctrladmission.Warnings, error) {
	return nil, nil
}

func NewPipelineVersionWebhook(client ctrlclient.Client) (http.Handler, error) {
	validating, err := ctrladmission.StandaloneWebhook(
		ctrladmission.WithCustomValidator(scheme, &k8sapi.PipelineVersion{}, &PipelineVersionsWebhook{Client: client}),
		ctrladmission.StandaloneOptions{},
	)
	if err != nil {
		return nil, err
	}

	return validating, nil
}
