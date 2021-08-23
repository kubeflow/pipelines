package compiler

import (
	"fmt"
	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8score "k8s.io/api/core/v1"
)

func (c *workflowCompiler) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	if component == nil {
		return fmt.Errorf("workflowCompiler.Importer: component spec must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	importerJson, err := marshaler.MarshalToString(importer)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Importer: marlshaling component spec to proto JSON failed: %w", err)
	}

	t := importerExecutorTemplate(importer, c.launcherImage)
	// TODO(Bobgy): how can we avoid template name collisions?
	importerTemplateName, err := c.addTemplate(t, name+"-importer")
	if err != nil {
		return err
	}
	wrapper := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramComponent, Default: wfapi.AnyStringPtr(componentJson)},
				{Name: paramImporter, Default: wfapi.AnyStringPtr(importerJson)},
			},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: []wfapi.DAGTask{
				{Name: "importer", Template: importerTemplateName, Arguments: wfapi.Arguments{
					Parameters: []wfapi.Parameter{
						{
							Name:  paramTask,
							Value: wfapi.AnyStringPtr(inputParameter(paramTask)),
						},
						{
							Name:  paramComponent,
							Value: wfapi.AnyStringPtr(inputParameter(paramComponent)),
						}, {
							Name:  paramImporter,
							Value: wfapi.AnyStringPtr(inputParameter(paramImporter)),
						},
					},
				}},
			},
		},
	}
	_, err = c.addTemplate(wrapper, name)
	return err
}

func importerExecutorTemplate(importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec, launcherImage string) *wfapi.Template {
	launcherArgs := []string{
		"--executor_type", "importer",
		"--task_spec", inputValue(paramTask),
		"--component_spec", inputValue(paramComponent),
		"--importer_spec", inputValue(paramImporter),
		"--pod_name",
		"$(KFP_POD_NAME)",
		"--pod_uid",
		"$(KFP_POD_UID)",
		"--mlmd_server_address", // METADATA_GRPC_SERVICE_* come from metadata-grpc-configmap
		"$(METADATA_GRPC_SERVICE_HOST)",
		"--mlmd_server_port",
		"$(METADATA_GRPC_SERVICE_PORT)",
	}
	mlmdConfigOptional := true
	return &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramComponent},
				{Name: paramImporter},
			},
		},
		Container: &k8score.Container{
			Image:   launcherImage,
			Command: []string{"launcher-v2"},
			Args:    launcherArgs,
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
			}},
			// TODO(Bobgy): support resource requests/limits
		},
	}
}
