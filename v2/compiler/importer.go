package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/component"
	k8score "k8s.io/api/core/v1"
)

func (c *workflowCompiler) Importer(name string, componentSpec *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	if componentSpec == nil {
		return fmt.Errorf("workflowCompiler.Importer: component spec must be non-nil")
	}
	componentJson, err := stablyMarshalJSON(componentSpec)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Importer: marshaling component spec to proto JSON failed: %w", err)
	}
	importerJson, err := stablyMarshalJSON(importer)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Importer: marlshaling importer spec to proto JSON failed: %w", err)
	}

	launcherArgs := []string{
		"--executor_type", "importer",
		"--task_spec", inputValue(paramTask),
		"--component_spec", inputValue(paramComponent),
		"--importer_spec", inputValue(paramImporter),
		"--pipeline_name", c.spec.PipelineInfo.GetName(),
		"--run_id", runID(),
		"--pod_name",
		fmt.Sprintf("$(%s)", component.EnvPodName),
		"--pod_uid",
		fmt.Sprintf("$(%s)", component.EnvPodUID),
		"--mlmd_server_address",
		fmt.Sprintf("$(%s)", component.EnvMetadataHost),
		"--mlmd_server_port",
		fmt.Sprintf("$(%s)", component.EnvMetadataPort),
	}
	importerTemplate := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramComponent, Default: wfapi.AnyStringPtr(componentJson)},
				{Name: paramImporter, Default: wfapi.AnyStringPtr(importerJson)},
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
	// TODO(Bobgy): how can we avoid template name collisions?
	_, err = c.addTemplate(importerTemplate, name)
	return err
}
