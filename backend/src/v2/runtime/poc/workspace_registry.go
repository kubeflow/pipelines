package poc

import (
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

func (c *Coordinator) workspaceProvisioner(run *model.Run, executorType string, reference *WorkspaceHandle) (WorkspaceProvisionerPlugin, error) {
	var kind string
	switch {
	case run != nil && run.WorkspaceProvisioner != nil:
		kind = *run.WorkspaceProvisioner
	case reference != nil && reference.Type != "":
		kind = reference.Type
	case executorType == ExecutorDocker:
		kind = WorkspaceTypeHostPath
	case executorType == ExecutorKubernetes:
		kind = WorkspaceTypePVC
	default:
		return nil, fmt.Errorf("no workspace provisioner selected for executor %q", executorType)
	}
	provisioner, ok := c.workspaceProvisioners[kind]
	if !ok {
		return nil, fmt.Errorf("workspace provisioner %q is not registered", kind)
	}
	return provisioner, nil
}
