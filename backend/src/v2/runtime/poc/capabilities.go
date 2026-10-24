package poc

import (
	"fmt"
	"slices"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (c *Coordinator) failBeforeDispatch(run *model.Run, manifest *RuntimeManifest, reason string) error {
	if root, ok := manifest.taskByScope("root"); ok {
		if _, err := c.updateTaskState(root.TaskID, apiv2.PipelineTask_FAILED, model.JSONData{"message": reason}, nil); err != nil {
			return err
		}
	}
	if manifest.ExecutorInfo == nil {
		manifest.ExecutorInfo = map[string]string{}
	}
	manifest.ExecutorInfo["error"] = reason
	encoded, err := manifest.ToJSON()
	if err != nil {
		return err
	}
	run.PipelineRuntimeManifest = model.LargeText(encoded)
	run.State, run.FinishedAtInSec = model.RuntimeStateFailed, c.time.Now().Unix()
	return c.updateRun(run)
}

// validatePipelineCapabilities checks every reachable branch before provisioning or execution.
func validatePipelineCapabilities(spec *pipelinespec.PipelineSpec, platform *pipelinespec.SinglePlatformSpec, capabilities ExecutorCapabilities, workspaceType string) error {
	if workspaceType != "" && !slices.Contains(capabilities.SupportedWorkspaceProvisioners, workspaceType) {
		return fmt.Errorf("executor does not support workspace provisioner %q; select a compatible executor or workspace", workspaceType)
	}
	visiting := map[*pipelinespec.ComponentSpec]bool{}
	var visit func(*pipelinespec.ComponentSpec, string) error
	visit = func(component *pipelinespec.ComponentSpec, path string) error {
		if component == nil {
			return fmt.Errorf("component for task %q is missing", path)
		}
		if visiting[component] {
			return fmt.Errorf("component cycle at task %q is not supported", path)
		}
		visiting[component] = true
		defer delete(visiting, component)
		if dag := component.GetDag(); dag != nil {
			for name, task := range dag.GetTasks() {
				if err := validateSupportedTask(task); err != nil {
					return fmt.Errorf("task %q: %w", path+"."+name, err)
				}
				if err := visit(spec.GetComponents()[task.GetComponentRef().GetName()], path+"."+name); err != nil {
					return err
				}
			}
			return nil
		}
		requirements, err := loadKubernetesExecutorConfig(component, platform)
		if err != nil {
			return fmt.Errorf("task %q has invalid execution requirements: %w", path, err)
		}
		if err := validatePlatformRequirements(requirements, capabilities); err != nil {
			return fmt.Errorf("task %q: %w", path, err)
		}
		importer, err := importerSpecForComponent(spec, component)
		if err != nil {
			return err
		}
		if importer != nil {
			return nil
		}
		container, err := containerForComponent(spec, component)
		if err != nil {
			return fmt.Errorf("task %q: %w", path, err)
		}
		if !capabilities.SupportsContainers {
			return fmt.Errorf("task %q requires container execution; select an executor declaring SupportsContainers", path)
		}
		if container.GetImage() == "" {
			return fmt.Errorf("task %q requires a container image", path)
		}
		if err := validateContainerCapabilities(container, capabilities); err != nil {
			return fmt.Errorf("task %q: %w", path, err)
		}
		return nil
	}
	return visit(spec.GetRoot(), "root")
}

func validateContainerCapabilities(container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, capabilities ExecutorCapabilities) error {
	if capabilities.SupportsKubernetes {
		return nil
	}
	if lifecycle := container.GetLifecycle(); lifecycle != nil && proto.Size(lifecycle) != 0 {
		return fmt.Errorf("container lifecycle hooks are unsupported by this executor; use the Kubernetes executor")
	}
	resources := container.GetResources()
	if resources == nil {
		return nil
	}
	if accelerator := resources.GetAccelerator(); accelerator != nil && proto.Size(accelerator) != 0 {
		return fmt.Errorf("accelerator resources are unsupported by this executor; use the Kubernetes executor")
	}
	for _, request := range []struct {
		value  string
		legacy float64
	}{
		{resources.GetResourceCpuRequest(), resources.GetCpuRequest()},       //nolint:staticcheck
		{resources.GetResourceMemoryRequest(), resources.GetMemoryRequest()}, //nolint:staticcheck
	} {
		if request.legacy != 0 {
			return fmt.Errorf("CPU or memory scheduling requests are unsupported by this executor; use limits or the Kubernetes executor")
		}
		if request.value != "" {
			quantity, err := resource.ParseQuantity(request.value)
			if err != nil || quantity.Sign() != 0 {
				return fmt.Errorf("CPU or memory scheduling requests are unsupported by this executor; use limits or the Kubernetes executor")
			}
		}
	}
	return nil
}

func validatePlatformRequirements(requirements *kubernetesplatform.KubernetesExecutorConfig, capabilities ExecutorCapabilities) error {
	if requirements == nil {
		return nil
	}
	if len(requirements.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown Kubernetes execution requirements are unsupported; update the runtime")
	}
	if capabilities.SupportsKubernetes {
		return nil
	}
	requirements = proto.Clone(requirements).(*kubernetesplatform.KubernetesExecutorConfig)
	if requirements.ImagePullPolicy == string(corev1.PullIfNotPresent) {
		requirements.ImagePullPolicy = ""
	}
	if hasExecutionRequirement(requirements.ProtoReflect()) {
		return fmt.Errorf("kubernetes execution requirements are unsupported; select an executor declaring SupportsKubernetes")
	}
	return nil
}
