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
	"strconv"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	k8score "k8s.io/api/core/v1"
)

func (c *workflowCompiler) Importer(name string, componentSpec *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	err := c.saveComponentSpec(name, componentSpec)
	if err != nil {
		return err
	}
	return c.saveComponentImpl(name, importer)
}

func (c *workflowCompiler) importerTask(name string, task *pipelinespec.PipelineTaskSpec, taskJSON string, parentDagID string, downloadToWorkspace bool) (*wfapi.DAGTask, error) {
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
		Template: c.addImporterTemplate(downloadToWorkspace, task.GetRetryPolicy()),
		Arguments: wfapi.Arguments{Parameters: append([]wfapi.Parameter{{
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
		}}, c.getTaskRetryParametersWithValues(task)...)},
	}, nil
}

// addImporterTemplate adds (or reuses) the importer template for the given
// downloadToWorkspace/retry combination. A task with a retry policy gets a
// "retry-" prefixed template carrying a retryStrategy, so importer tasks
// without a retry policy keep sharing the plain template unaffected.
func (c *workflowCompiler) addImporterTemplate(downloadToWorkspace bool, taskRetrySpec *pipelinespec.PipelineTaskSpec_RetryPolicy) string {
	name := "system-importer"
	if downloadToWorkspace {
		name += "-workspace"
	}
	if taskRetrySpec != nil {
		name = "retry-" + name
	}
	if _, alreadyExists := c.templates[name]; alreadyExists {
		return name
	}
	args := []string{
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
		"--mlmd_server_address", metadata.GetMetadataConfig().Address,
		"--mlmd_server_port", metadata.GetMetadataConfig().Port,
	}
	args = append(args,
		"--cache_disabled="+strconv.FormatBool(c.cacheDisabled),
		"--ml_pipeline_tls_enabled="+strconv.FormatBool(c.mlPipelineTLSEnabled),
		"--metadata_tls_enabled="+strconv.FormatBool(common.GetMetadataTLSEnabled()),
	)

	// Always passed; empty unless a custom CA bundle is configured.
	caCertPath := ""
	setCABundle := false
	if common.GetCaBundleSecretName() != "" || common.GetCaBundleConfigMapName() != "" {
		caCertPath = common.CustomCaCertPath
		setCABundle = true
	}
	args = append(args, "--ca_cert_path", caCertPath)

	args = append(args, "--log_level", pipelineLogLevelArg(), "--publish_logs", publishLogsArg())

	var volumeMounts []k8score.VolumeMount
	var volumes []k8score.Volume
	if downloadToWorkspace {
		volumeMounts = append(volumeMounts, k8score.VolumeMount{
			Name:      workspaceVolumeName,
			MountPath: component.WorkspaceMountPath,
		})
		volumes = append(volumes, k8score.Volume{
			Name: workspaceVolumeName,
			VolumeSource: k8score.VolumeSource{
				PersistentVolumeClaim: &k8score.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("{{workflow.name}}-%s", workspaceVolumeName),
				},
			},
		})
	}

	inputParameters := []wfapi.Parameter{
		{Name: paramTask},
		{Name: paramComponent},
		{Name: paramImporter},
		{Name: paramParentDagID},
	}
	if taskRetrySpec != nil {
		inputParameters = append(inputParameters,
			wfapi.Parameter{Name: paramRetryMaxCount},
			wfapi.Parameter{Name: paramRetryBackOffDuration},
			wfapi.Parameter{Name: paramRetryBackOffFactor},
			wfapi.Parameter{Name: paramRetryBackOffMaxDuration},
		)
	}

	importerTemplate := &wfapi.Template{
		Name:   name,
		Inputs: wfapi.Inputs{Parameters: inputParameters},
		Container: &k8score.Container{
			Image:        c.launcherImage,
			Command:      c.launcherCommand,
			Args:         args,
			EnvFrom:      []k8score.EnvFromSource{metadataEnvFrom},
			Env:          commonEnvs,
			Resources:    driverResources,
			VolumeMounts: volumeMounts,
		},
		Volumes: volumes,
	}
	if taskRetrySpec != nil {
		importerTemplate.RetryStrategy = c.getTaskRetryStrategyFromInput(
			inputParameter(paramRetryMaxCount),
			inputParameter(paramRetryBackOffDuration),
			inputParameter(paramRetryBackOffFactor),
			inputParameter(paramRetryBackOffMaxDuration),
		)
	}

	setRuntimeRole(importerTemplate, util.ExecutionRuntimeRoleLauncher)
	// If TLS is enabled (apiserver or metadata), add the custom CA bundle to the importer template.
	if setCABundle {
		ConfigureCustomCABundle(importerTemplate)
	}
	applySecurityContextToTemplate(importerTemplate)
	c.templates[name] = importerTemplate
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *importerTemplate)
	return name
}
