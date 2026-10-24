package poc

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

type registeredWorkspace struct {
	WorkspaceProvisionerPlugin
	kind string
}

func (p *registeredWorkspace) Type() string { return p.kind }

func TestRegisteredWorkspaceProvisionerIsUsedForSelectionAndRecovery(t *testing.T) {
	coordinator := NewCoordinator(nil, nil, nil, nil, nil, nil, nil, false)
	host := &registeredWorkspace{kind: WorkspaceTypeHostPath}
	coordinator.RegisterWorkspaceProvisioner(host)
	selected, err := coordinator.workspaceProvisioner(&model.Run{}, ExecutorDocker, nil)
	require.NoError(t, err)
	require.Same(t, host, selected)
	custom := &registeredWorkspace{kind: "custom-workspace"}
	coordinator.RegisterWorkspaceProvisioner(custom)
	selected, err = coordinator.workspaceProvisioner(&model.Run{
		RunDetails: model.RunDetails{WorkspaceProvisioner: util.StringPointer(custom.Type())},
	}, ExecutorKubernetes, &WorkspaceHandle{Type: WorkspaceTypePVC})
	require.NoError(t, err)
	require.Same(t, custom, selected, "persisted provisioning selection must win over current executor defaults")
	_, err = coordinator.workspaceProvisioner(&model.Run{
		RunDetails: model.RunDetails{WorkspaceProvisioner: util.StringPointer("missing")},
	}, ExecutorKubernetes, nil)
	require.ErrorContains(t, err, "not registered")
}
