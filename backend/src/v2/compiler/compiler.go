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

package main

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	pb "github.com/kubeflow/pipelines/api/v2alpha1/go"
	"github.com/kubeflow/pipelines/backend/src/v2/common"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/templates"
	"github.com/pkg/errors"
)

const (
	rootDagDriverTaskName = "driver-kfp-root"
)

const (
	templateNameExecutorDriver    = "kfp-executor-driver"
	templateNameDagDriver         = "kfp-dag-driver"
	templateNameExecutorPublisher = "kfp-executor-publisher"
)

func CompilePipelineSpec(
	pipelineSpec *pb.PipelineSpec,
	deploymentConfig *pb.PipelineDeploymentConfig,
) (*workflowapi.Workflow, error) {

	// validation
	if pipelineSpec.GetPipelineInfo().GetName() == "" {
		return nil, errors.New("Name is empty")
	}

	// initialization
	var workflow workflowapi.Workflow
	workflow.APIVersion = "argoproj.io/v1alpha1"
	workflow.Kind = "Workflow"
	workflow.GenerateName = pipelineSpec.GetPipelineInfo().GetName() + "-"

	spec, err := generateSpec(pipelineSpec, deploymentConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to generate workflow spec")
	}
	workflow.Spec = *spec

	return &workflow, nil
}

func generateSpec(
	pipelineSpec *pb.PipelineSpec,
	deploymentConfig *pb.PipelineDeploymentConfig,
) (*workflowapi.WorkflowSpec, error) {
	tasks := pipelineSpec.GetTasks()
	var spec workflowapi.WorkflowSpec

	// generate helper templates
	executorDriver := templates.Driver(false)
	executorDriver.Name = templateNameExecutorDriver
	dagDriver := templates.Driver(true)
	dagDriver.Name = templateNameDagDriver
	executorPublisher := templates.Publisher(common.PublisherType_EXECUTOR)
	executorPublisher.Name = templateNameExecutorPublisher
	executorTemplates := templates.Executor(templateNameExecutorDriver, templateNameExecutorPublisher)

	// generate root template
	var root workflowapi.Template
	root.Name = "kfp-root"
	rootDag := initRootDag(&spec, templateNameDagDriver)
	root.DAG = rootDag
	// TODO: make a generic default value
	defaultTaskSpec := `{"taskInfo":{"name":"hello-world-dag"},"inputs":{"parameters":{"text":{"runtimeValue":{"constantValue":{"stringValue":"Hello, World!"}}}}}}`

	spec.Arguments.Parameters = []workflowapi.Parameter{
		{Name: "task-spec", Value: v1alpha1.AnyStringPtr(defaultTaskSpec)},
	}

	subDag, err := templates.Dag(&templates.DagArgs{
		Tasks:                &tasks,
		DeploymentConfig:     deploymentConfig,
		ExecutorTemplateName: templates.TemplateNameExecutor,
	})
	if err != nil {
		return nil, err
	}
	parentContextName := "{{tasks." + rootDagDriverTaskName + ".outputs.parameters." + templates.DriverParamContextName + "}}"
	root.DAG.Tasks = append(root.DAG.Tasks, workflowapi.DAGTask{
		Name:         "sub-dag",
		Template:     subDag.Name,
		Dependencies: []string{rootDagDriverTaskName},
		Arguments: workflowapi.Arguments{
			Parameters: []workflowapi.Parameter{
				{Name: templates.DagParamContextName, Value: v1alpha1.AnyStringPtr(parentContextName)},
			},
		},
	})

	spec.Templates = []workflowapi.Template{root, *subDag, *executorDriver, *dagDriver, *executorPublisher}
	for _, template := range executorTemplates {
		spec.Templates = append(spec.Templates, *template)
	}
	spec.Entrypoint = root.Name
	return &spec, nil
}

func initRootDag(spec *workflowapi.WorkflowSpec, templateNameDagDriver string) *workflowapi.DAGTemplate {
	root := &workflowapi.DAGTemplate{}
	// TODO(Bobgy): shall we pass a lambda "addTemplate()" here instead?
	driverTask := &workflowapi.DAGTask{}
	driverTask.Name = rootDagDriverTaskName
	driverTask.Template = templateNameDagDriver
	rootExecutionName := "kfp-root-{{workflow.name}}"
	workflowParameterTaskSpec := "{{workflow.parameters.task-spec}}"
	driverType := "DAG"
	parentContextName := "" // root has no parent
	driverTask.Arguments.Parameters = []workflowapi.Parameter{
		{Name: templates.DriverParamExecutionName, Value: v1alpha1.AnyStringPtr(rootExecutionName)},
		{Name: templates.DriverParamTaskSpec, Value: v1alpha1.AnyStringPtr(workflowParameterTaskSpec)},
		{Name: templates.DriverParamDriverType, Value: v1alpha1.AnyStringPtr(driverType)},
		{Name: templates.DriverParamParentContextName, Value: v1alpha1.AnyStringPtr(parentContextName)},
	}
	root.Tasks = append(root.Tasks, *driverTask)
	return root
}
