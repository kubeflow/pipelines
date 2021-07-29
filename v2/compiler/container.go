package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8score "k8s.io/api/core/v1"
)

const (
	containerLauncherImage = "gcr.io/gongyuan-dev/dev/kfp-launcher-v2:latest"
	volumePathKFPLauncher  = "/kfp-launcher"
	volumeNameKFPLauncher  = "kfp-launcher"
)

func (c *workflowCompiler) Container(name string, component *pipelinespec.ComponentSpec, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	if component == nil {
		return fmt.Errorf("workflowCompiler.Container: component spec must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Container: marlshaling component spec to proto JSON failed: %w", err)
	}
	driverTask, driverOutputs := c.containerDriverTask("driver", componentJson, inputParameter(paramTask), inputParameter(paramDAGContextID), inputParameter(paramDAGExecutionID))
	if err != nil {
		return err
	}
	t := containerExecutorTemplate(container)
	// TODO(Bobgy): how can we avoid template name collisions?
	containerTemplateName, err := c.addTemplate(t, name+"-container")
	if err != nil {
		return err
	}
	wrapper := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramDAGContextID},
				{Name: paramDAGExecutionID},
			},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: []wfapi.DAGTask{
				*driverTask,
				{Name: "container", Template: containerTemplateName, Dependencies: []string{driverTask.Name}, Arguments: wfapi.Arguments{
					Parameters: []wfapi.Parameter{{
						Name:  paramExecutorInput,
						Value: wfapi.AnyStringPtr(driverOutputs.executorInput),
					}, {
						Name:  paramExecutionID,
						Value: wfapi.AnyStringPtr(driverOutputs.executionID),
					}},
				}},
			},
		},
	}
	_, err = c.addTemplate(wrapper, name)
	return err
}

type containerDriverOutputs struct {
	executorInput string
	executionID   string
}

func (c *workflowCompiler) containerDriverTask(name, component, task, dagContextID, dagExecutionID string) (*wfapi.DAGTask, *containerDriverOutputs) {
	dagTask := &wfapi.DAGTask{
		Name:     name,
		Template: c.addContainerDriverTemplate(),
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent, Value: wfapi.AnyStringPtr(component)},
				{Name: paramTask, Value: wfapi.AnyStringPtr(task)},
				{Name: paramDAGContextID, Value: wfapi.AnyStringPtr(dagContextID)},
				{Name: paramDAGExecutionID, Value: wfapi.AnyStringPtr(dagExecutionID)},
			},
		},
	}
	outputs := &containerDriverOutputs{
		executorInput: taskOutputParameter(name, paramExecutorInput),
		executionID:   taskOutputParameter(name, paramExecutionID),
	}
	return dagTask, outputs
}

func (c *workflowCompiler) addContainerDriverTemplate() string {
	name := "system-container-driver"
	_, ok := c.templates[name]
	if ok {
		return name
	}
	t := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent},
				{Name: paramTask},
				{Name: paramDAGContextID},
				{Name: paramDAGExecutionID},
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramExecutionID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/execution-id"}},
				{Name: paramExecutorInput, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/executor-input"}},
			},
		},
		Container: &k8score.Container{
			Image:   c.driverImage,
			Command: []string{"/bin/kfp/driver"},
			Args: []string{
				"--type", "CONTAINER",
				"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
				"--run_id", runID(),
				"--dag_context_id", inputValue(paramDAGContextID),
				"--dag_execution_id", inputValue(paramDAGExecutionID),
				"--component", inputValue(paramComponent),
				"--task", inputValue(paramTask),
				"--execution_id_path", outputPath(paramExecutionID),
				"--executor_input_path", outputPath(paramExecutorInput),
			},
		},
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
}

func containerExecutorTemplate(container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) *wfapi.Template {
	userCmdArgs := make([]string, 0, len(container.Command)+len(container.Args))
	userCmdArgs = append(userCmdArgs, container.Command...)
	userCmdArgs = append(userCmdArgs, container.Args...)
	launcherCmd := []string{
		volumePathKFPLauncher + "/launch",
		"--execution_id", inputValue(paramExecutionID),
		"--executor_input", inputValue(paramExecutorInput),
		"--namespace",
		"$(KFP_NAMESPACE)",
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
	return &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramExecutorInput},
				{Name: paramExecutionID},
			},
		},
		Volumes: []k8score.Volume{{
			Name: volumeNameKFPLauncher,
			VolumeSource: k8score.VolumeSource{
				EmptyDir: &k8score.EmptyDirVolumeSource{},
			},
		}},
		InitContainers: []wfapi.UserContainer{{
			Container: k8score.Container{
				Name:    "kfp-launcher",
				Image:   containerLauncherImage,
				Command: []string{"mount_launcher.sh"},
				VolumeMounts: []k8score.VolumeMount{{
					Name:      volumeNameKFPLauncher,
					MountPath: volumePathKFPLauncher,
				}},
				ImagePullPolicy: "Always",
			},
		}},
		Container: &k8score.Container{
			Command: launcherCmd,
			Args:    userCmdArgs,
			Image:   container.Image,
			VolumeMounts: []k8score.VolumeMount{{
				Name:      volumeNameKFPLauncher,
				MountPath: volumePathKFPLauncher,
			}},
			EnvFrom: []k8score.EnvFromSource{{
				ConfigMapRef: &k8score.ConfigMapEnvSource{
					LocalObjectReference: k8score.LocalObjectReference{
						Name: "metadata-grpc-configmap",
					},
					Optional: &mlmdConfigOptional,
				},
			}},
			Env: []k8score.EnvVar{{
				Name: "KFP_NAMESPACE",
				ValueFrom: &k8score.EnvVarSource{
					FieldRef: &k8score.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			}, {
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
			}},
			// TODO(Bobgy): support resource requests/limits
		},
	}
}
