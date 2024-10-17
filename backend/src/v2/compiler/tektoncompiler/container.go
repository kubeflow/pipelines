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
	"encoding/json"
	"fmt"
	"os"

	"github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/apis/kfptask"
	ktv1alpha1 "github.com/kubeflow/kfp-tekton/tekton-catalog/tekton-kfptask/pkg/apis/kfptask/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	volumeNameKFPLauncher   = "kfp-launcher"
	kfpLauncherPath         = "/tekton/home/launch"
	MetadataGRPCServiceHost = "metadata-grpc-service.kubeflow.svc.cluster.local"
	MetadataGPRCServicePort = "8080"
	MLPipelineServiceHost   = "ml-pipeline.kubeflow.svc.cluster.local"
	MLPipelineServicePort   = "8887"
	LauncherImage           = "gcr.io/ml-pipeline/kfp-launcher@sha256:50151a8615c8d6907aa627902dce50a2619fd231f25d1e5c2a72737a2ea4001e"
	MinioServiceHost        = "minio-service.kubeflow.svc.cluster.local"
	MinioServicePort        = "9000"
)

var (
	envVarInit              = false
	metadataGRPCServiceHost = MetadataGRPCServiceHost
	metadataGRPCServicePort = MetadataGPRCServicePort
	mlPipelineServiceHost   = MLPipelineServiceHost
	mlPipelineServicePort   = MLPipelineServicePort
	launcherImage           = LauncherImage
	minioServiceHost        = MinioServiceHost
	minioServicePort        = MinioServicePort
)

func initEnvVars() {
	// fill in the Env vars we support
	// assuming:
	// 1. MLMD is deployed in the same namespace as ml-pipeline
	// 2. using `ml-pipeline` and `metadata-grpc-service` as service names
	mlPipelineServiceHost = os.Getenv("ML_PIPELINE_SERVICE_HOST")
	if mlPipelineServiceHost == "" {
		mlPipelineServiceHost = MLPipelineServiceHost
	}
	mlPipelineServicePort = os.Getenv("ML_PIPELINE_SERVICE_PORT_GRPC")
	if mlPipelineServicePort == "" {
		mlPipelineServicePort = MLPipelineServicePort
	}
	metadataGRPCServiceHost = os.Getenv("METADATA_GRPC_SERVICE_SERVICE_HOST")
	if metadataGRPCServiceHost == "" {
		metadataGRPCServiceHost = MetadataGRPCServiceHost
	}
	metadataGRPCServicePort = os.Getenv("METADATA_GRPC_SERVICE_SERVICE_PORT")
	if metadataGRPCServicePort == "" {
		metadataGRPCServicePort = MetadataGPRCServicePort
	}
	launcherImage = os.Getenv("V2_LAUNCHER_IMAGE")
	if launcherImage == "" {
		launcherImage = LauncherImage
	}
	minioServiceHost = os.Getenv("MINIO_SERVICE_SERVICE_HOST")
	if minioServiceHost == "" {
		minioServiceHost = MinioServiceHost
	}
	minioServicePort = os.Getenv("MINIO_SERVICE_SERVICE_PORT")
	if minioServicePort == "" {
		minioServicePort = MinioServicePort
	}
	envVarInit = true
}

func GetMLMDHost() string {
	if !envVarInit {
		initEnvVars()
	}
	return metadataGRPCServiceHost
}

func GetMLMDPort() string {
	if !envVarInit {
		initEnvVars()
	}
	return metadataGRPCServicePort
}

func GetMLPipelineHost() string {
	if !envVarInit {
		initEnvVars()
	}
	return mlPipelineServiceHost
}

func GetMLPipelinePort() string {
	if !envVarInit {
		initEnvVars()
	}
	return mlPipelineServicePort
}

func GetLauncherImage() string {
	if !envVarInit {
		initEnvVars()
	}
	return launcherImage
}

func GetMinioHost() string {
	if !envVarInit {
		initEnvVars()
	}
	return minioServiceHost
}

func GetMinioPort() string {
	if !envVarInit {
		initEnvVars()
	}
	return minioServicePort
}

// add KubernetesSpec for the container of the component
func (c *pipelinerunCompiler) AddKubernetesSpec(name string, kubernetesSpec *structpb.Struct) error {
	err := c.saveKubernetesSpec(name, kubernetesSpec)
	if err != nil {
		return err
	}
	return nil
}

func (c *pipelinerunCompiler) Container(taskName, compRef string,
	task *pipelinespec.PipelineTaskSpec,
	component *pipelinespec.ComponentSpec,
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
) error {

	err := c.saveComponentSpec(compRef, component)
	if err != nil {
		return err
	}
	err = c.saveComponentImpl(compRef, container)
	if err != nil {
		return err
	}

	componentSpec, err := c.useComponentSpec(compRef)
	if err != nil {
		return fmt.Errorf("component spec for %q not found", compRef)
	}
	taskSpecJson, err := stablyMarshalJSON(task)
	if err != nil {
		return err
	}
	containerImpl, err := c.useComponentImpl(compRef)
	if err != nil {
		return err
	}

	exitHandler := false
	if task.GetTriggerPolicy().GetStrategy().String() == "ALL_UPSTREAM_TASKS_COMPLETED" {
		exitHandler = true
	}
	kubernetesConfigPlaceholder, _ := c.useKubernetesImpl(taskName)
	return c.containerDriverTask(taskName, &containerDriverInputs{
		component:        componentSpec,
		task:             taskSpecJson,
		container:        containerImpl,
		parentDag:        c.CurrentDag(),
		taskDef:          task,
		containerDef:     container,
		exitHandler:      exitHandler,
		kubernetesConfig: kubernetesConfigPlaceholder,
		inLoopDag:        c.HasLoopName(c.CurrentDag()),
	})
}

type containerDriverOutputs struct {
	// break down podSpecPath to the following
	executionId    string
	executiorInput string
	cached         string
	condition      string
	podSpecPatch   string
}

type containerDriverInputs struct {
	component        string
	task             string
	taskDef          *pipelinespec.PipelineTaskSpec
	container        string
	containerDef     *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	parentDag        string
	iterationIndex   string // optional, when this is an iteration task
	exitHandler      bool
	kubernetesConfig string
	inLoopDag        bool
}

func (i *containerDriverInputs) getParentDagID(isExitHandler bool) string {
	if i.parentDag == "" {
		return "0"
	}
	if isExitHandler && i.parentDag == compiler.RootComponentName {
		return fmt.Sprintf("$(params.%s)", paramParentDagID)
	} else if i.inLoopDag {
		return fmt.Sprintf("$(params.%s)", paramNameDagExecutionId)
	} else {
		return taskOutputParameter(getDAGDriverTaskName(i.parentDag), paramExecutionID)
	}
}

func (i *containerDriverInputs) getParentDagCondition(isExitHandler bool) string {
	if i.parentDag == "" {
		return "0"
	}
	if isExitHandler && i.parentDag == compiler.RootComponentName {
		return fmt.Sprintf("$(params.%s)", paramCondition)
	} else {
		return taskOutputParameter(getDAGDriverTaskName(i.parentDag), paramCondition)
	}
}

func (c *pipelinerunCompiler) containerDriverTask(name string, inputs *containerDriverInputs) error {

	t, err := c.containerExecutorTemplate(name, inputs.containerDef, c.spec.PipelineInfo.GetName())

	if err != nil {
		return err
	}
	driverTask := &pipelineapi.PipelineTask{
		Name:     name,
		TaskSpec: t,
		Params: []pipelineapi.Param{
			// "--type", "CONTAINER",
			{
				Name:  paramNameType,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: "CONTAINER"},
			},
			// "--pipeline-name", c.spec.GetPipelineInfo().GetName(),
			{
				Name:  paramNamePipelineName,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: c.spec.GetPipelineInfo().GetName()},
			},
			// "--run-id", runID(),
			{
				Name:  paramRunId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: runID()},
			},
			// "--dag-execution-id"
			{
				Name:  paramNameDagExecutionId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.getParentDagID(c.ExitHandlerScope())},
			},
			// "--task"
			{
				Name:  paramTask,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.task},
			},
			// "--container"
			{
				Name:  paramContainer,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.container},
			},
			// "--iteration-index", inputValue(paramIterationIndex),
			{
				Name:  paramNameIterationIndex,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.iterationIndex},
			},
			// "--kubernetes-config"
			{
				Name:  paramKubernetesConfig,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.kubernetesConfig},
			},
			// "--mlmd-server-address"
			{
				Name:  paramNameMLMDServerHost,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDHost()},
			},
			// "--mlmd-server-port"
			{
				Name:  paramNameMLMDServerPort,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDPort()},
			},
			// "--component"
			{
				Name:  paramComponent,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.component},
			},
			// produce the following outputs:
			// - execution-id
			// - condition
		},
	}

	if len(inputs.taskDef.GetDependentTasks()) > 0 {
		driverTask.RunAfter = inputs.taskDef.GetDependentTasks()
	}

	// adding WhenExpress for condition only if the task belongs to a DAG had a condition TriggerPolicy
	if c.ConditionScope() {
		driverTask.When = pipelineapi.WhenExpressions{
			pipelineapi.WhenExpression{
				Input:    inputs.getParentDagCondition(c.ExitHandlerScope()),
				Operator: selection.NotIn,
				Values:   []string{"false"},
			},
		}
	}

	c.addPipelineTask(driverTask)

	return nil
}

func (c *pipelinerunCompiler) containerExecutorTemplate(
	name string, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
	pipelineName string,
) (*pipelineapi.EmbeddedTask, error) {
	userCmdArgs := make([]string, 0, len(container.Command)+len(container.Args))
	userCmdArgs = append(userCmdArgs, container.Command...)
	userCmdArgs = append(userCmdArgs, container.Args...)
	// userCmdArgs = append(userCmdArgs, "--executor_input", "{{$}}", "--function_to_execute", inputValue(paramFunctionToExecute))
	launcherCmd := []string{
		kfpLauncherPath,
		"--pipeline_name", pipelineName,
		"--run_id", inputValue(paramRunId),
		"--execution_id", inputValue(paramExecutionID),
		"--executor_input", inputValue(paramExecutorInput),
		"--component_spec", inputValue(paramComponent),
		"--pod_name",
		"$(KFP_POD_NAME)",
		"--pod_uid",
		"$(KFP_POD_UID)",
		"--mlmd_server_address", // METADATA_GRPC_SERVICE_* come from metadata-grpc-configmap
		"$(METADATA_GRPC_SERVICE_HOST)",
		"--mlmd_server_port",
		"$(METADATA_GRPC_SERVICE_PORT)",
		"--", // separater before user command and args
	}
	mlmdConfigOptional := true
	kfpTaskSpec := ktv1alpha1.KfpTaskSpec{
		TaskSpec: &pipelineapi.TaskSpec{
			Params: []pipelineapi.ParamSpec{
				{Name: paramExecutorInput, Type: "string"}, // --executor_input
				{Name: paramExecutionID, Type: "string"},   // --execution_id
				{Name: paramRunId, Type: "string"},         // --run_id
				{Name: paramComponent, Type: "string"},     // --component
			},
			Steps: []pipelineapi.Step{
				// step 1: copy launcher
				{
					Name:            "kfp-launcher",
					Image:           c.launcherImage,
					Command:         []string{"launcher-v2", "--copy", kfpLauncherPath},
					ImagePullPolicy: "Always",
				},
				// wrap user program with executor
				{
					Name:    "user-main",
					Image:   container.Image,
					Command: launcherCmd,
					Args:    userCmdArgs,
					EnvFrom: []k8score.EnvFromSource{{
						ConfigMapRef: &k8score.ConfigMapEnvSource{
							LocalObjectReference: k8score.LocalObjectReference{
								Name: "metadata-grpc-configmap",
							},
							Optional: &mlmdConfigOptional,
						},
					}},
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
					}, {
						// override the k8s envs for the following two envs
						// to make sure launcher can find the ml-pipeline host properly
						Name:  "ML_PIPELINE_SERVICE_HOST",
						Value: GetMLPipelineHost(),
					}, {
						Name:  "ML_PIPELINE_SERVICE_PORT_GRPC",
						Value: GetMLPipelinePort(),
					}, {
						Name:  "MINIO_SERVICE_SERVICE_HOST",
						Value: GetMinioHost(),
					}, {
						Name:  "MINIO_SERVICE_SERVICE_PORT",
						Value: GetMinioPort(),
					}},
				},
			},
		},
	}

	raw, err := json.Marshal(kfpTaskSpec)
	if err != nil {
		return nil, fmt.Errorf("unable to Marshal KfpTaskSpec:%v", err)
	}

	return &pipelineapi.EmbeddedTask{
		Metadata: pipelineapi.PipelineTaskMetadata{
			Annotations: map[string]string{
				"pipelines.kubeflow.org/v2_pipeline": "true",
			},
			Labels: map[string]string{
				"pipelines.kubeflow.org/v2_component": "true",
			},
		},
		TypeMeta: runtime.TypeMeta{
			Kind:       kfptask.Kind,
			APIVersion: ktv1alpha1.SchemeGroupVersion.String(),
		},
		Spec: runtime.RawExtension{
			Raw: raw,
		},
	}, nil
}
