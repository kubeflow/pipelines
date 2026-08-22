// Copyright 2026 The Kubeflow Authors
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
	"os"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	k8score "k8s.io/api/core/v1"
)

func (c *workflowCompiler) TriggerPipeline(name string, componentSpec *pipelinespec.ComponentSpec, trigger *pipelinespec.PipelineDeploymentConfig_TriggerPipelineSpec) error {
	err := c.saveComponentSpec(name, componentSpec)
	if err != nil {
		return err
	}
	return c.saveComponentImpl(name, trigger)
}

func (c *workflowCompiler) triggerPipelineTask(name string, task *pipelinespec.PipelineTaskSpec, taskJSON string, parentDagID string) (*wfapi.DAGTask, error) {
	componentPlaceholder, err := c.useComponentSpec(task.GetComponentRef().GetName())
	if err != nil {
		return nil, err
	}
	triggerPlaceholder, err := c.useComponentImpl(task.GetComponentRef().GetName())
	if err != nil {
		return nil, err
	}
	return &wfapi.DAGTask{
		Name:     name,
		Template: c.addTriggerPipelineTemplate(),
		Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{{
			Name:  paramTask,
			Value: wfapi.AnyStringPtr(taskJSON),
		}, {
			Name:  paramComponent,
			Value: wfapi.AnyStringPtr(componentPlaceholder),
		}, {
			Name:  paramTriggerPipeline,
			Value: wfapi.AnyStringPtr(triggerPlaceholder),
		}, {
			Name:  paramParentDagID,
			Value: wfapi.AnyStringPtr(parentDagID),
		}}},
	}, nil
}

func (c *workflowCompiler) addTriggerPipelineTemplate() string {
	name := "system-trigger-pipeline"
	if _, alreadyExists := c.templates[name]; alreadyExists {
		return name
	}
	args := []string{
		"--executor_type", "trigger_pipeline",
		"--task_spec", inputValue(paramTask),
		"--component_spec", inputValue(paramComponent),
		"--trigger_pipeline_spec", inputValue(paramTriggerPipeline),
		"--pipeline_name", c.spec.PipelineInfo.GetName(),
		"--run_id", runID(),
		"--parent_dag_id", inputValue(paramParentDagID),
		"--pod_name",
		fmt.Sprintf("$(%s)", component.EnvPodName),
		"--pod_uid",
		fmt.Sprintf("$(%s)", component.EnvPodUID),
		"--mlmd_server_address", metadata.GetMetadataConfig().Address,
		"--mlmd_server_port", metadata.GetMetadataConfig().Port,
		"--ml_pipeline_server_address", config.GetMLPipelineServerConfig().Address,
		"--ml_pipeline_server_port", config.GetMLPipelineServerConfig().Port,
	}
	if c.cacheDisabled {
		args = append(args, "--cache_disabled")
	}
	if c.mlPipelineTLSEnabled {
		args = append(args, "--ml_pipeline_tls_enabled")
	}
	if common.GetMetadataTLSEnabled() {
		args = append(args, "--metadata_tls_enabled")
	}

	setCABundle := false
	if common.GetCaBundleSecretName() != "" || common.GetCaBundleConfigMapName() != "" {
		args = append(args, "--ca_cert_path", common.CustomCaCertPath)
		setCABundle = true
	}

	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		args = append(args, "--log_level", value)
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		args = append(args, "--publish_logs", value)
	}

	triggerTemplate := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramComponent},
				{Name: paramTriggerPipeline},
				{Name: paramParentDagID},
			},
		},
		Container: &k8score.Container{
			Image:     c.launcherImage,
			Command:   c.launcherCommand,
			Args:      args,
			EnvFrom:   []k8score.EnvFromSource{metadataEnvFrom},
			Env:       commonEnvs,
			Resources: driverResources,
		},
	}

	setRuntimeRole(triggerTemplate, util.ExecutionRuntimeRoleLauncher)
	if setCABundle {
		ConfigureCustomCABundle(triggerTemplate)
	}
	applySecurityContextToTemplate(triggerTemplate)
	c.templates[name] = triggerTemplate
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *triggerTemplate)
	return name
}
