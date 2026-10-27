package poc

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

// materializedRetryDependencies resolves IR edges within each actual parent/iteration.
func materializedRetryDependencies(tasks []*model.Task, spec *pipelinespec.PipelineSpec) (map[string][]string, error) {
	children := map[string]map[string][]*model.Task{}
	var roots []*model.Task
	for _, task := range tasks {
		if task.ParentTaskUUID == nil {
			if task.ScopePath == "root" {
				roots = append(roots, task)
			}
			continue
		}
		parent := *task.ParentTaskUUID
		if children[parent] == nil {
			children[parent] = map[string][]*model.Task{}
		}
		children[parent][task.Name] = append(children[parent][task.Name], task)
	}
	if len(roots) != 1 {
		return nil, fmt.Errorf("run retry requires exactly one materialized root task")
	}
	dependencies := map[string][]string{}
	visited := map[string]bool{}
	var walk func(string, *pipelinespec.ComponentSpec) error
	walk = func(parent string, component *pipelinespec.ComponentSpec) error {
		if component == nil {
			return fmt.Errorf("retry task %s has no persisted IR component", parent)
		}
		if visited[parent] {
			return fmt.Errorf("retry task hierarchy contains a cycle at %s", parent)
		}
		visited[parent] = true
		if component.GetDag() == nil {
			return nil
		}
		for name, taskSpec := range component.GetDag().GetTasks() {
			childComponent := spec.GetComponents()[taskSpec.GetComponentRef().GetName()]
			for _, task := range children[parent][name] {
				for _, dependency := range taskSpec.GetDependentTasks() {
					for _, source := range children[parent][dependency] {
						dependencies[task.UUID] = append(dependencies[task.UUID], source.UUID)
					}
				}
				if taskSpec.GetParameterIterator() != nil {
					for _, iterations := range children[task.UUID] {
						for _, iteration := range iterations {
							if err := walk(iteration.UUID, childComponent); err != nil {
								return err
							}
						}
					}
				} else if err := walk(task.UUID, childComponent); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(roots[0].UUID, spec.GetRoot()); err != nil {
		return nil, err
	}
	return dependencies, nil
}
