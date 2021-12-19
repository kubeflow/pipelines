package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/component"
	k8score "k8s.io/api/core/v1"
)

const (
	volumeNameKFPLauncher = "kfp-launcher"
)

func (c *workflowCompiler) Container(name string, component *pipelinespec.ComponentSpec, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	if component == nil {
		return fmt.Errorf("workflowCompiler.Container: component spec must be non-nil")
	}
	componentJson, err := stablyMarshalJSON(component)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Container: marlshaling component spec to proto JSON failed: %w", err)
	}
	containerJson, err := stablyMarshalJSON(container)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Container: marlshaling pipeline container spec to proto JSON failed: %w", err)
	}
	driver, driverOutputs := c.containerDriverTask(
		"driver",
		containerDriverInputs{
			component:      inputParameter(paramComponent),
			task:           inputParameter(paramTask),
			container:      containerJson,
			dagExecutionID: inputParameter(paramDAGExecutionID),
			iterationIndex: inputParameter(paramIterationIndex),
		},
	)
	executor := c.containerExecutorTask(
		"container",
		driverOutputs.podSpecPatch,
	)
	// TODO(Bobgy): can we add dependencies automatically
	executor.Dependencies = []string{driver.Name}
	executor.When = driverOutputs.cached + " != true"

	// TODO(Bobgy): reuse the entire 2-step container template
	wrapper := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramDAGExecutionID},
				{Name: paramComponent, Default: wfapi.AnyStringPtr(componentJson)},
				{Name: paramIterationIndex, Default: wfapi.AnyStringPtr("-1")},
			},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: []wfapi.DAGTask{
				*driver,
				*executor,
			},
		},
	}
	_, err = c.addTemplate(wrapper, name)
	return err
}

type containerDriverOutputs struct {
	podSpecPatch string
	cached       string
}

type containerDriverInputs struct {
	component      string
	task           string
	container      string
	dagExecutionID string
	iterationIndex string // optional, when this is an iteration task
}

func (c *workflowCompiler) containerDriverTask(name string, inputs containerDriverInputs) (*wfapi.DAGTask, *containerDriverOutputs) {
	dagTask := &wfapi.DAGTask{
		Name:     name,
		Template: c.addContainerDriverTemplate(),
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent, Value: wfapi.AnyStringPtr(inputs.component)},
				{Name: paramTask, Value: wfapi.AnyStringPtr(inputs.task)},
				{Name: paramContainer, Value: wfapi.AnyStringPtr(inputs.container)},
				{Name: paramDAGExecutionID, Value: wfapi.AnyStringPtr(inputs.dagExecutionID)},
				{Name: paramIterationIndex, Value: wfapi.AnyStringPtr(inputs.iterationIndex)},
			},
		},
	}
	outputs := &containerDriverOutputs{
		podSpecPatch: taskOutputParameter(name, paramPodSpecPatch),
		cached:       taskOutputParameter(name, paramCachedDecision),
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
				{Name: paramContainer},
				{Name: paramDAGExecutionID},
				{Name: paramIterationIndex},
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramPodSpecPatch, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/pod-spec-patch", Default: wfapi.AnyStringPtr("")}},
				{Name: paramCachedDecision, Default: wfapi.AnyStringPtr("false"), ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/cached-decision", Default: wfapi.AnyStringPtr("false")}},
			},
		},
		Container: &k8score.Container{
			Image:   c.driverImage,
			Command: []string{"driver"},
			Args: []string{
				"--type", "CONTAINER",
				"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
				"--run_id", runID(),
				"--dag_execution_id", inputValue(paramDAGExecutionID),
				"--component", inputValue(paramComponent),
				"--task", inputValue(paramTask),
				"--container", inputValue(paramContainer),
				"--iteration_index", inputValue(paramIterationIndex),
				"--cached_decision_path", outputPath(paramCachedDecision),
				"--pod_spec_patch_path", outputPath(paramPodSpecPatch),
			},
			Resources: driverResources,
		},
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
}

// containerExecutorTask returns an argo workflows DAGTask.
// name: argo workflows DAG task name
// podSpecPatch: argo workflows DAG parameter, it can be either a string or a placeholder
func (c *workflowCompiler) containerExecutorTask(name string, podSpecPatch string) *wfapi.DAGTask {
	return &wfapi.DAGTask{
		Name:     name,
		Template: c.addContainerExecutorTemplate(),
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{
				{Name: paramPodSpecPatch, Value: wfapi.AnyStringPtr(podSpecPatch)},
			},
		},
	}
}

// addContainerExecutorTemplate adds a generic container executor template for
// any container component task.
// During runtime, it's expected that pod-spec-patch will specify command, args
// and resources etc, that are different for different tasks.
func (c *workflowCompiler) addContainerExecutorTemplate() string {
	name := "system-container-executor"
	_, ok := c.templates[name]
	if ok {
		return name
	}
	t := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramPodSpecPatch},
			},
		},
		// PodSpecPatch input param is where actual image, command and
		// args come from. It is treated as a strategic merge patch on
		// top of the Pod spec.
		PodSpecPatch: inputValue(paramPodSpecPatch),
		Volumes: []k8score.Volume{{
			Name: volumeNameKFPLauncher,
			VolumeSource: k8score.VolumeSource{
				EmptyDir: &k8score.EmptyDirVolumeSource{},
			},
		}},
		InitContainers: []wfapi.UserContainer{{
			Container: k8score.Container{
				Name:    "kfp-launcher",
				Image:   c.launcherImage,
				Command: []string{"launcher-v2", "--copy", component.KFPLauncherPath},
				VolumeMounts: []k8score.VolumeMount{{
					Name:      volumeNameKFPLauncher,
					MountPath: component.VolumePathKFPLauncher,
				}},
				Resources: launcherResources,
			},
		}},
		Container: &k8score.Container{
			// The placeholder image and command should always be
			// overridden in podSpecPatch.
			// In case we have a bug, the placeholder image is kept
			// in gcr.io/ml-pipeline, so that we are sure the image
			// never exists.
			// These are added to pass argo workflows linting.
			Image:   "gcr.io/ml-pipeline/should-be-overridden-during-runtime",
			Command: []string{"should-be-overridden-during-runtime"},
			VolumeMounts: []k8score.VolumeMount{{
				Name:      volumeNameKFPLauncher,
				MountPath: component.VolumePathKFPLauncher,
			}},
			EnvFrom: []k8score.EnvFromSource{metadataEnvFrom},
			Env:     commonEnvs,
		},
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
}
