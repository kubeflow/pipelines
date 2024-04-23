// Copyright 2023 The Kubeflow Authors
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

package tektoncompiler

import (
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	k8score "k8s.io/api/core/v1"
)

func (c *pipelinerunCompiler) Importer(name string,
	task *pipelinespec.PipelineTaskSpec,
	componentSpec *pipelinespec.ComponentSpec,
	importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec,
) error {

	err := c.saveComponentSpec(name, componentSpec)
	if err != nil {
		return err
	}

	componentSpecStr, err := c.useComponentSpec(name)
	if err != nil {
		return err
	}

	if err := c.saveComponentImpl(name, importer); err != nil {
		return err
	}

	componentImplStr, err := c.useComponentImpl(name)
	if err != nil {
		return err
	}

	taskSpecJson, err := stablyMarshalJSON(task)
	if err != nil {
		return err
	}

	launcherArgs := []string{
		"--executor_type", "importer",
		"--task_spec", inputValue(paramTask),
		"--component_spec", inputValue(paramComponent),
		"--importer_spec", inputValue(paramImporter),
		"--pipeline_name", c.spec.PipelineInfo.GetName(),
		"--run_id", inputValue(paramNameRunId),
		"--parent_dag_id", inputValue(paramParentDagID),
		"--pod_name",
		"$(KFP_POD_NAME)",
		"--pod_uid",
		"$(KFP_POD_UID)",
		"--mlmd_server_address",
		"$(METADATA_GRPC_SERVICE_HOST)",
		"--mlmd_server_port",
		"$(METADATA_GRPC_SERVICE_PORT)",
	}

	pipelineTask := &pipelineapi.PipelineTask{
		Name: name,
		TaskSpec: &pipelineapi.EmbeddedTask{
			Metadata: pipelineapi.PipelineTaskMetadata{
				Annotations: map[string]string{
					"pipelines.kubeflow.org/v2_pipeline": "true",
				},
				Labels: map[string]string{
					"pipelines.kubeflow.org/v2_component": "true",
				},
			},
			TaskSpec: pipelineapi.TaskSpec{
				Params: []pipelineapi.ParamSpec{
					{Name: paramTask, Type: "string"},
					{Name: paramComponent, Type: "string"},
					{Name: paramImporter, Type: "string"},
					{Name: paramParentDagID, Type: "string"},
				},
				Results: []pipelineapi.TaskResult{
					{Name: paramExecutionID, Description: "execution id"},
					{Name: paramExecutorInput, Description: "executor input"},
				},
				Steps: []pipelineapi.Step{
					{
						Name:    "importer-main",
						Image:   c.launcherImage,
						Command: []string{"launcher-v2"},
						Args:    launcherArgs,
						Env: []k8score.EnvVar{{
							Name: "KFP_POD_NAME",
							ValueFrom: &k8score.EnvVarSource{
								FieldRef: &k8score.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						}, {
							Name: "KFP_POD_UID",
							ValueFrom: &k8score.EnvVarSource{
								FieldRef: &k8score.ObjectFieldSelector{
									FieldPath: "metadata.uid",
								},
							},
						}, {
							Name:  "METADATA_GRPC_SERVICE_HOST",
							Value: GetMLMDHost(),
						}, {
							Name:  "METADATA_GRPC_SERVICE_PORT",
							Value: GetMLMDPort(),
						}},
					},
				},
			},
		},
		RunAfter: append(task.GetDependentTasks(), getDAGDriverTaskName(c.CurrentDag())),
		Params: []pipelineapi.Param{
			{
				Name:  paramTask,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: taskSpecJson},
			},
			{
				Name:  paramComponent,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: componentSpecStr},
			},
			{
				Name:  paramImporter,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: componentImplStr},
			},
			{
				Name:  paramParentDagID,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: taskOutputParameter(getDAGDriverTaskName(c.CurrentDag()), paramExecutionID)},
			},
			{
				Name:  paramNameRunId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: runID()},
			},
		},
	}
	c.addPipelineTask(pipelineTask)

	return nil
}
