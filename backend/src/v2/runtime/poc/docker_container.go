package poc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/api/resource"
)

var dockerEnvReference = regexp.MustCompile(`\$\$|\$\([A-Za-z_][A-Za-z0-9_]*\)`)
var dockerEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func resolveDockerValue(value string, input *pipelinespec.ExecutorInput) (string, error) {
	_, resolved, err := component.CompileCommandAndArgs(input, []string{"docker"}, []string{value})
	if err != nil {
		return "", err
	}
	return resolved[0], nil
}

func expandDockerEnvironment(value string, environment map[string]string) string {
	return dockerEnvReference.ReplaceAllStringFunc(value, func(reference string) string {
		if reference == "$$" {
			return "$"
		}
		if replacement, exists := environment[reference[2:len(reference)-1]]; exists {
			return replacement
		}
		return reference
	})
}

func dockerEnvironmentArgs(request TaskExecutionRequest) ([]string, map[string]string, error) {
	args := dockerProxyEnvArgs()
	environment := map[string]string{}
	for index := 1; index < len(args); index += 2 {
		key, value, _ := strings.Cut(args[index], "=")
		environment[key] = value
	}
	if request.ArtifactLocalRoot != "" {
		environment["ARTIFACT_LOCAL_PATH"] = request.ArtifactLocalRoot
	}
	for _, variable := range request.Container.GetEnv() {
		if !dockerEnvName.MatchString(variable.GetName()) {
			return nil, nil, fmt.Errorf("docker container environment variable has an invalid name")
		}
		value, err := resolveDockerValue(variable.GetValue(), request.ExecutorInput)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot resolve Docker environment variable %q: %w", variable.GetName(), err)
		}
		value = expandDockerEnvironment(value, environment)
		environment[variable.GetName()] = value
		args = append(args, "-e", variable.GetName()+"="+value)
	}
	if request.ArtifactLocalRoot != "" {
		args = append(args, "-e", "ARTIFACT_LOCAL_PATH="+request.ArtifactLocalRoot)
		environment["ARTIFACT_LOCAL_PATH"] = request.ArtifactLocalRoot
	}
	return args, environment, nil
}

func dockerResourceArgs(container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, input *pipelinespec.ExecutorInput) ([]string, error) {
	if lifecycle := container.GetLifecycle(); lifecycle != nil && proto.Size(lifecycle) != 0 {
		return nil, fmt.Errorf("docker does not support container lifecycle hooks")
	}
	resources := container.GetResources()
	if resources == nil {
		return nil, nil
	}
	if accelerator := resources.GetAccelerator(); accelerator != nil && proto.Size(accelerator) != 0 {
		return nil, fmt.Errorf("docker does not support accelerator resource requirements")
	}
	// Standalone Docker has limits but cannot guarantee Kubernetes-style scheduling reservations.
	for _, request := range []struct {
		value  string
		legacy float64
	}{
		{resources.GetResourceCpuRequest(), resources.GetCpuRequest()},       //nolint:staticcheck
		{resources.GetResourceMemoryRequest(), resources.GetMemoryRequest()}, //nolint:staticcheck
	} {
		quantity, err := dockerResourceQuantity(request.value, request.legacy, false, input)
		if err != nil {
			return nil, err
		}
		if quantity != nil && quantity.Sign() != 0 {
			return nil, fmt.Errorf("docker does not support CPU or memory scheduling requests; use limits or the Kubernetes executor")
		}
	}
	args := []string{}
	for _, limit := range []struct {
		value  string
		legacy float64
		memory bool
		flag   string
	}{
		{resources.GetResourceCpuLimit(), resources.GetCpuLimit(), false, "--cpus"},        //nolint:staticcheck
		{resources.GetResourceMemoryLimit(), resources.GetMemoryLimit(), true, "--memory"}, //nolint:staticcheck
	} {
		quantity, err := dockerResourceQuantity(limit.value, limit.legacy, limit.memory, input)
		if err != nil {
			return nil, err
		}
		if quantity == nil || quantity.Sign() == 0 {
			continue
		}
		value := strconv.FormatFloat(quantity.AsApproximateFloat64(), 'f', -1, 64)
		if limit.memory {
			value = strconv.FormatInt(quantity.Value(), 10)
		}
		args = append(args, limit.flag, value)
	}
	return args, nil
}

func dockerResourceQuantity(value string, legacy float64, memory bool, input *pipelinespec.ExecutorInput) (*resource.Quantity, error) {
	if value == "" && legacy == 0 {
		return nil, nil
	}
	if value == "" {
		value = strconv.FormatFloat(legacy, 'f', -1, 64)
		if memory {
			value += "G"
		}
	}
	resolved, err := resolveDockerValue(value, input)
	if err != nil {
		return nil, err
	}
	quantity, err := resource.ParseQuantity(resolved)
	if err != nil {
		return nil, fmt.Errorf("invalid Docker resource quantity %q: %w", resolved, err)
	}
	if quantity.Sign() < 0 {
		return nil, fmt.Errorf("docker resource quantities must not be negative")
	}
	return &quantity, nil
}
