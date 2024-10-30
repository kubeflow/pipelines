// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"strconv"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	k8score "k8s.io/api/core/v1"
)

func (c *workflowCompiler) Importer(name string, componentSpec *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	err := c.saveComponentSpec(name, componentSpec)
	if err != nil {
		return err
	}
	return c.saveComponentImpl(name, importer)
}

func (c *workflowCompiler) importerTask(name string, task *pipelinespec.PipelineTaskSpec, taskJSON string, parentDagID string) (*wfapi.DAGTask, error) {
	componentPlaceholder, err := c.useComponentSpec(task.GetComponentRef().GetName())
	if err != nil {
		return nil, err
	}
	importerPlaceholder, err := c.useComponentImpl(task.GetComponentRef().GetName())
	if err != nil {
		return nil, err
	}
	return &wfapi.DAGTask{
		Name:     name,
		Template: c.addImporterTemplate(),
		Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{{
			Name:  paramTask,
			Value: wfapi.AnyStringPtr(taskJSON),
		}, {
			Name:  paramComponent,
			Value: wfapi.AnyStringPtr(componentPlaceholder),
		}, {
			Name:  paramImporter,
			Value: wfapi.AnyStringPtr(importerPlaceholder),
		}, {
			Name:  paramParentDagID,
			Value: wfapi.AnyStringPtr(parentDagID),
		}}},
	}, nil
}

func (c *workflowCompiler) addImporterTemplate() string {
	name := "system-importer"
	if _, alreadyExists := c.templates[name]; alreadyExists {
		return name
	}
	launcherArgs := []string{
		"--executor_type", "importer",
		"--task_spec", inputValue(paramTask),
		"--component_spec", inputValue(paramComponent),
		"--importer_spec", inputValue(paramImporter),
		"--pipeline_name", c.spec.PipelineInfo.GetName(),
		"--run_id", runID(),
		"--parent_dag_id", inputValue(paramParentDagID),
		"--pod_name",
		fmt.Sprintf("$(%s)", component.EnvPodName),
		"--pod_uid",
		fmt.Sprintf("$(%s)", component.EnvPodUID),
		"--mlmd_server_address", common.GetMetadataGrpcServiceServiceHost(),
		"--mlmd_server_port", common.GetMetadataGrpcServiceServicePort(),
		"--metadataTLSEnabled", strconv.FormatBool(common.GetMetadataTLSEnabled()),
		"--ca_cert_path", common.GetCaCertPath(),
	}
	importerTemplate := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramComponent},
				{Name: paramImporter},
				{Name: paramParentDagID},
			},
		},
		Container: &k8score.Container{
			Image:     c.launcherImage,
			Command:   []string{"launcher-v2"},
			Args:      launcherArgs,
			EnvFrom:   []k8score.EnvFromSource{metadataEnvFrom},
			Env:       commonEnvs,
			Resources: driverResources,
		},
	}

	ConfigureCABundle(importerTemplate)

	c.templates[name] = importerTemplate
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *importerTemplate)
	return name
}
