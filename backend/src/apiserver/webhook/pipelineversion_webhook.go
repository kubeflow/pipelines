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
	"fmt"
	"net/http"

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
	Client        ctrlclient.Client
	ClientNoCache ctrlclient.Client
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

func (p *PipelineVersionsWebhook) getPipeline(ctx context.Context, namespace string, name string) (*k8sapi.Pipeline, error) {
	pipeline := &k8sapi.Pipeline{}
	nsName := types.NamespacedName{Namespace: namespace, Name: name}
	err := p.Client.Get(ctx, nsName, pipeline)
	if err == nil {
		return pipeline, nil
	}

	if !apierrors.IsNotFound(err) {
		return nil, newBadRequestError(fmt.Sprintf("Failed to get the Pipeline %s/%s: %v", namespace, name, err))
	}

	// Fallback to not using the cache
	err = p.ClientNoCache.Get(ctx, nsName, pipeline)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newBadRequestError("The spec.pipelineName doesn't map to an existing Pipeline object")
		}

		return nil, newBadRequestError(fmt.Sprintf("Failed to get the Pipeline %s/%s: %v", namespace, name, err))
	}

	return pipeline, nil
}

func (p *PipelineVersionsWebhook) ValidateCreate(
	ctx context.Context, obj runtime.Object,
) (ctrladmission.Warnings, error) {
	pipelineVersion, ok := obj.(*k8sapi.PipelineVersion)
	if !ok {
		return nil, newBadRequestError(fmt.Sprintf("Expected a PipelineVersion object but got %T", pipelineVersion))
	}

	modelPipelineVersion, err := pipelineVersion.ToModel()
	if err != nil {
		return nil, newBadRequestError(fmt.Sprintf("The pipeline spec is invalid: %v", err))
	}

	// cache enabled or not doesn't matter in this context
	tmpl, err := template.NewV2SpecTemplate([]byte(modelPipelineVersion.PipelineSpec), false, nil)
	if err != nil {
		return nil, newBadRequestError(fmt.Sprintf("The pipeline spec is invalid: %v", err))
	}

	err = common.ValidatePipelineName(tmpl.V2PipelineName())
	if err != nil {
		return nil, newBadRequestError(err.Error())
	}

	return nil, nil
}

func (p *PipelineVersionsWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (ctrladmission.Warnings, error) {
	oldPipelineVersion, ok := oldObj.(*k8sapi.PipelineVersion)
	if !ok {
		return nil, newBadRequestError(fmt.Sprintf("Expected a PipelineVersion but got %T", oldObj))
	}

	newPipelineVersion, ok := newObj.(*k8sapi.PipelineVersion)
	if !ok {
		return nil, newBadRequestError(fmt.Sprintf("Expected a PipelineVersion but got %T", newObj))
	}

	if oldPipelineVersion.Generation != newPipelineVersion.Generation {
		return nil, newBadRequestError("Pipeline spec is immutable; only metadata changes (labels/annotations) are allowed")
	}

	return nil, nil
}

// ValidateDelete is unused but required to implement the ctrladmission.CustomValidator interface.
func (p *PipelineVersionsWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (ctrladmission.Warnings, error) {
	return nil, nil
}

func (p *PipelineVersionsWebhook) Default(ctx context.Context, obj runtime.Object) error {
	pipelineVersion, ok := obj.(*k8sapi.PipelineVersion)
	if !ok {
		return &apierrors.StatusError{
			ErrStatus: metav1.Status{
				Code:    http.StatusBadRequest,
				Reason:  metav1.StatusReasonBadRequest,
				Message: fmt.Sprintf("expected a PipelineVersion object but got %T", pipelineVersion),
			},
		}
	}

	pipeline, err := p.getPipeline(ctx, pipelineVersion.Namespace, pipelineVersion.Spec.PipelineName)
	if err != nil {
		return err
	}

	if pipelineVersion.Labels == nil {
		pipelineVersion.Labels = map[string]string{}
	}

	// Labels for efficient querying
	pipelineVersion.Labels["pipelines.kubeflow.org/pipeline-id"] = string(pipeline.UID)
	pipelineVersion.Labels["pipelines.kubeflow.org/pipeline"] = pipeline.Name

	trueVal := true

	for i := range pipelineVersion.OwnerReferences {
		ownerRef := &pipelineVersion.OwnerReferences[i]
		if ownerRef.APIVersion != k8sapi.GroupVersion.String() || ownerRef.Kind != "Pipeline" {
			continue
		}

		ownerRef.Name = pipeline.Name
		ownerRef.BlockOwnerDeletion = &trueVal
		ownerRef.UID = pipeline.UID

		return nil
	}

	pipelineVersion.OwnerReferences = append(pipelineVersion.OwnerReferences, metav1.OwnerReference{
		APIVersion:         k8sapi.GroupVersion.String(),
		Kind:               "Pipeline",
		Name:               pipeline.Name,
		BlockOwnerDeletion: &trueVal,
		UID:                pipeline.UID,
	})

	return nil
}

// NewPipelineVersionWebhook returns the validating webhook and mutating webhook HTTP handlers
func NewPipelineVersionWebhook(
	client ctrlclient.Client, clientNoCache ctrlclient.Client,
) (http.Handler, http.Handler, error) {
	validating, err := ctrladmission.StandaloneWebhook(
		ctrladmission.WithCustomValidator(
			scheme, &k8sapi.PipelineVersion{}, &PipelineVersionsWebhook{Client: client, ClientNoCache: clientNoCache},
		),
		ctrladmission.StandaloneOptions{},
	)
	if err != nil {
		return nil, nil, err
	}

	mutating, err := ctrladmission.StandaloneWebhook(
		ctrladmission.WithCustomDefaulter(
			scheme, &k8sapi.PipelineVersion{}, &PipelineVersionsWebhook{Client: client, ClientNoCache: clientNoCache},
		),
		ctrladmission.StandaloneOptions{},
	)
	if err != nil {
		return nil, nil, err
	}

	return validating, mutating, nil
}
