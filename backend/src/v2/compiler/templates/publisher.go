// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templates

import (
	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	k8sv1 "k8s.io/api/core/v1"
)

// TODO(Bobgy): make image configurable
// gcr.io/gongyuan-pipeline-test/kfp-driver:latest
const (
	publisherImage     = "gcr.io/gongyuan-pipeline-test/kfp-publisher"
	publisherImageRef  = "@sha256:3623052f6bd4f02da6199d1971ec948b96fb00c6d8c61725d3cfda998b304efe"
	publisherImageFull = publisherImage + publisherImageRef
)

const (
	// Inputs
	PublisherParamExecutionId   = paramPrefixKfpInternal + "execution-id"
	PublisherParamPublisherType = paramPrefixKfpInternal + "publisher-type"
	PublisherParamOutputsSpec   = paramPrefixKfpInternal + "outputs-spec"
	PublisherArtifactParameters = paramPrefixKfpInternal + "parameters"
	// Paths
	publisherInputPathParameters = "/kfp/inputs/parameters"
)

// TODO(Bobgy): parameters is no longer needed.
// TODO(Bobgy): reuse existing templates if they are the same.
func Publisher(publisherType string) *workflowapi.Template {
	publisher := &workflowapi.Template{}
	// publisher.Name is not set, it should be set after calling this method.
	publisher.Container = &k8sv1.Container{}
	publisher.Container.Image = publisherImageFull
	publisher.Container.Command = []string{"/bin/kfp-publisher"}
	publisher.Inputs.Parameters = []workflowapi.Parameter{
		{Name: PublisherParamExecutionId},
		{Name: PublisherParamOutputsSpec},
		{Name: PublisherParamPublisherType},
	}
	publisher.Inputs.Artifacts = []workflowapi.Artifact{{
		Name:     PublisherArtifactParameters,
		Path:     publisherInputPathParameters,
		Optional: true,
	}}
	publisher.Container.Args = []string{
		"--logtostderr",
		// TODO(Bobgy): make this configurable
		"--mlmd_url=metadata-grpc-service.kubeflow.svc.cluster.local:8080",
		"--execution_id={{inputs.parameters." + PublisherParamExecutionId + "}}",
		"--publisher_type={{inputs.parameters." + PublisherParamPublisherType + "}}",
		"--component_outputs_spec={{inputs.parameters." + PublisherParamOutputsSpec + "}}",
		"--input_path_parameters={{inputs.artifacts." + PublisherArtifactParameters + ".path}}",
	}
	return publisher
}
