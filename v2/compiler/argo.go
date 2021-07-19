package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Compile(job *pipelinespec.PipelineJob) (*wfapi.Workflow, error) {
	spec, err := getPipelineSpec(job)
	if err != nil {
		return nil, err
	}
	// validation
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, fmt.Errorf("pipelineInfo.name is empty")
	}

	// initialization
	wf := &wfapi.Workflow{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			GenerateName: spec.GetPipelineInfo().GetName() + "-",
			// TODO(Bobgy): uncomment the following. Temporarily commented for development purposes.
			// Annotations: map[string]string{
			// 	"pipelines.kubeflow.org/v2_pipeline": "true",
			// },
		},
		Spec: wfapi.WorkflowSpec{
			PodMetadata: &wfapi.Metadata{
				Annotations: map[string]string{
					"pipelines.kubeflow.org/v2_component": "true",
				},
				Labels: map[string]string{
					"pipelines.kubeflow.org/v2_component": "true",
				},
			},
			ServiceAccountName: "pipeline-runner",
			Entrypoint:         rootComponentName,
		},
	}
	compiler := &workflowCompiler{
		wf:          wf,
		templates:   make(map[string]*wfapi.Template),
		driverImage: "gcr.io/gongyuan-dev/dev/kfp-driver:latest",
		job:         job,
		spec:        spec,
	}

	// compile
	Accept(job, compiler)

	return compiler.wf, nil
}

type workflowCompiler struct {
	// inputs
	job  *pipelinespec.PipelineJob
	spec *pipelinespec.PipelineSpec
	// state
	wf          *wfapi.Workflow
	templates   map[string]*wfapi.Template
	driverImage string
}

func (c *workflowCompiler) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	return nil
}
func (c *workflowCompiler) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	return nil
}
func (c *workflowCompiler) DAG(name string, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) error {
	if name != "root" {
		return fmt.Errorf("SubDAG not implemented yet")
	}
	dag := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramDAGContextID},
				{Name: paramDAGExecutionID},
			},
		},
		DAG: &wfapi.DAGTemplate{},
	}
	for _, kfpTask := range dagSpec.GetTasks() {
		marshaler := jsonpb.Marshaler{}
		taskJson, err := marshaler.MarshalToString(kfpTask)
		if err != nil {
			return fmt.Errorf("DAG: marshaling task spec to proto JSON failed: %w", err)
		}
		dag.DAG.Tasks = append(dag.DAG.Tasks, wfapi.DAGTask{
			Name:     kfpTask.GetTaskInfo().GetName(),
			Template: c.templateName(kfpTask.GetComponentRef().GetName()),
			Arguments: wfapi.Arguments{
				Parameters: []wfapi.Parameter{
					{
						Name:  paramDAGContextID,
						Value: wfapi.AnyStringPtr(inputParameter(paramDAGContextID)),
					},
					{
						Name:  paramDAGExecutionID,
						Value: wfapi.AnyStringPtr(inputParameter(paramDAGExecutionID)),
					},
					{
						Name:  paramTask,
						Value: wfapi.AnyStringPtr(taskJson),
					},
				},
			},
			Dependencies: []string{},
		})
	}
	// TODO(Bobgy): how can we avoid template name collisions?
	dagName, err := c.addTemplate(dag, name+"-dag")
	task := &pipelinespec.PipelineTaskSpec{}
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if name == "root" {
		// runtime config is input to the entire pipeline (root DAG)
		runtimeConfig = c.job.GetRuntimeConfig()
	}
	driverTask, outputs, err := c.dagDriverTask("driver", componentSpec, task, runtimeConfig)
	if err != nil {
		return err
	}
	wrapper := &wfapi.Template{}
	wrapper.DAG = &wfapi.DAGTemplate{
		Tasks: []wfapi.DAGTask{
			*driverTask,
			{
				Name: "dag", Template: dagName, Dependencies: []string{"driver"},
				Arguments: wfapi.Arguments{
					Parameters: []wfapi.Parameter{
						{Name: paramDAGExecutionID, Value: wfapi.AnyStringPtr(outputs.executionID)},
						{Name: paramDAGContextID, Value: wfapi.AnyStringPtr(outputs.contextID)},
					},
				},
			},
		},
	}
	_, err = c.addTemplate(wrapper, name)
	return err
}

var errAlreadyExists = fmt.Errorf("template already exists")

func (c *workflowCompiler) addTemplate(t *wfapi.Template, name string) (string, error) {
	t.Name = c.templateName(name)
	_, ok := c.templates[t.Name]
	if ok {
		return "", fmt.Errorf("template name=%q: %w", t.Name, errAlreadyExists)
	}
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	c.templates[t.Name] = t
	return t.Name, nil
}

const (
	paramComponent      = "component"      // component spec
	paramTask           = "task"           // task spec
	paramRuntimeConfig  = "runtime-config" // job runtime config, pipeline level inputs
	paramDAGContextID   = "dag-context-id"
	paramDAGExecutionID = "dag-execution-id"
	paramExecutionID    = "execution-id"
	paramContextID      = "context-id"
	paramExecutorInput  = "executor-input"
)

type dagDriverOutputs struct {
	contextID, executionID string
}

func (c *workflowCompiler) dagDriverTask(name string, component *pipelinespec.ComponentSpec, task *pipelinespec.PipelineTaskSpec, runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig) (*wfapi.DAGTask, *dagDriverOutputs, error) {
	if component == nil {
		return nil, nil, fmt.Errorf("dagDriverTask: component must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return nil, nil, fmt.Errorf("dagDriverTask: marlshaling component spec to proto JSON failed: %w", err)
	}
	taskJson := "{}"
	if task != nil {
		taskJson, err = marshaler.MarshalToString(task)
		if err != nil {
			return nil, nil, fmt.Errorf("dagDriverTask: marshaling task spec to proto JSON failed: %w", err)
		}
	}
	runtimeConfigJson := "{}"
	if runtimeConfig != nil {
		runtimeConfigJson, err = marshaler.MarshalToString(runtimeConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("dagDriverTask: marshaling runtime config to proto JSON failed: %w", err)
		}
	}
	templateName := c.addDAGDriverTemplate()
	t := &wfapi.DAGTask{
		Name:     name,
		Template: templateName,
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{
				{
					Name:  paramComponent,
					Value: wfapi.AnyStringPtr(componentJson),
				},
				{
					Name:  paramTask,
					Value: wfapi.AnyStringPtr(taskJson),
				},
				{
					Name:  paramRuntimeConfig,
					Value: wfapi.AnyStringPtr(runtimeConfigJson),
				},
			},
		},
	}
	return t, &dagDriverOutputs{
		contextID:   taskOutputParameter(name, paramContextID),
		executionID: taskOutputParameter(name, paramExecutionID),
	}, nil
}

func (c *workflowCompiler) addDAGDriverTemplate() string {
	name := "system-dag-driver"
	_, ok := c.templates[name]
	if ok {
		return name
	}
	t := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent},
				{Name: paramRuntimeConfig},
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramExecutionID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/execution-id"}},
				{Name: paramContextID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/context-id"}},
			},
		},
		Container: &k8score.Container{
			Image:   c.driverImage,
			Command: []string{"/bin/kfp/driver"},
			Args: []string{
				"--type", "ROOT_DAG",
				"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
				"--run_id", runID(),
				"--component", inputValue(paramComponent),
				"--runtime_config", inputValue(paramRuntimeConfig),
				"--execution_id_path", outputPath(paramExecutionID),
				"--context_id_path", outputPath(paramContextID),
			},
		},
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
}

func runID() string {
	// KFP API server converts this to KFP run ID.
	return "{{workflow.uid}}"
}

func workflowParameter(name string) string {
	return fmt.Sprintf("{{workflow.parameters.%s}}", name)
}

func inputValue(parameter string) string {
	return fmt.Sprintf("{{inputs.parameters.%s}}", parameter)
}

func inputParameter(parameter string) string {
	return fmt.Sprintf("{{inputs.parameters.%s}}", parameter)
}

func outputPath(parameter string) string {
	return fmt.Sprintf("{{outputs.parameters.%s.path}}", parameter)
}

func taskOutputParameter(task string, param string) string {
	return fmt.Sprintf("{{tasks.%s.outputs.parameters.%s}}", task, param)
}

func (c *workflowCompiler) templateName(componentName string) string {
	// TODO(Bobgy): sanitize component name, because argo template names
	// must be valid Kubernetes resource names.
	return componentName
}
