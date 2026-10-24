package poc

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestHostWorkspaceProvisionResolveAndConfinedCleanup(t *testing.T) {
	root := t.TempDir()
	previous := viper.Get(common.V2RuntimeWorkspaceRoot)
	viper.Set(common.V2RuntimeWorkspaceRoot, root)
	t.Cleanup(func() { viper.Set(common.V2RuntimeWorkspaceRoot, previous) })
	provisioner := hostPathWorkspaceProvisioner{}
	request := WorkspaceRequest{RunID: "run"}
	workspace, err := provisioner.Provision(context.Background(), request)
	require.NoError(t, err)
	same, err := provisioner.Provision(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, workspace, same)
	_, err = provisioner.ResolveAttachment(context.Background(), workspace, ExecutionTarget{Executor: ExecutorDocker})
	require.NoError(t, err)
	_, err = provisioner.ResolveAttachment(context.Background(), workspace, ExecutionTarget{Executor: ExecutorKubernetes})
	require.Error(t, err)
	artifact := filepath.Join(root, "run", "artifact-local", "published")
	require.NoError(t, os.MkdirAll(filepath.Dir(artifact), 0o755))
	require.NoError(t, os.WriteFile(artifact, []byte("published"), 0o600))
	forged := &WorkspaceHandle{Version: 1, Type: WorkspaceTypeHostPath, Metadata: map[string]string{"runID": "run", "path": filepath.Dir(artifact)}}
	require.ErrorContains(t, provisioner.Cleanup(context.Background(), forged), "derived")
	require.NoError(t, provisioner.Cleanup(context.Background(), workspace))
	require.NoError(t, provisioner.Cleanup(context.Background(), workspace))
	assert.FileExists(t, artifact)
	_, err = provisioner.Provision(context.Background(), WorkspaceRequest{RunID: "../outside"})
	require.Error(t, err)
	other := t.TempDir()
	require.NoError(t, os.Symlink(other, filepath.Join(root, "symlink-run")))
	_, err = provisioner.Provision(context.Background(), WorkspaceRequest{RunID: "symlink-run"})
	require.ErrorContains(t, err, "symlink")
}

func TestHostWorkspaceCleanupRepairsOnlyContainerOwnedWorkspaceDirectories(t *testing.T) {
	root := t.TempDir()
	previous := viper.Get(common.V2RuntimeWorkspaceRoot)
	viper.Set(common.V2RuntimeWorkspaceRoot, root)
	t.Cleanup(func() { viper.Set(common.V2RuntimeWorkspaceRoot, previous) })
	provisioner := hostPathWorkspaceProvisioner{}
	workspace, err := provisioner.Provision(context.Background(), WorkspaceRequest{RunID: "run"})
	require.NoError(t, err)
	path := workspace.Metadata["path"]
	data := filepath.Join(path, "data")
	require.NoError(t, os.MkdirAll(data, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(data, "test_file.txt"), []byte("container output"), 0o600))
	require.NoError(t, os.Chmod(data, 0o555))
	t.Cleanup(func() { _ = os.Chmod(data, 0o755) })
	published := filepath.Join(root, "run", "artifact-local", "published")
	require.NoError(t, os.MkdirAll(published, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(published, "artifact"), []byte("published"), 0o600))
	require.NoError(t, os.Symlink(published, filepath.Join(path, "published-link")))
	calls, removals := 0, 0
	provisioner.removeAll = func(candidate string) error {
		assert.Equal(t, path, candidate)
		removals++
		if removals == 1 {
			return &os.PathError{Op: "unlinkat", Path: data, Err: os.ErrPermission}
		}
		return os.RemoveAll(candidate)
	}
	provisioner.command = func(ctx context.Context, args ...string) (string, string, error) {
		calls++
		deadline, bounded := ctx.Deadline()
		require.True(t, bounded)
		assert.LessOrEqual(t, time.Until(deadline), 45*time.Second)
		assert.Equal(t, []string{
			"run", "--rm", "--network", "none", "--read-only", "--user", "0:0",
			"--cap-drop", "ALL", "--cap-add", "FOWNER", "--cap-add", "DAC_OVERRIDE",
			"--security-opt", "no-new-privileges", "--pids-limit", "16", "--memory", "64m", "--cpus", "0.25",
			"--volume", bindMount(path, "/workspace"),
			"alpine:3.23", "timeout", "-s", "KILL", "30",
			"find", "/workspace", "-xdev", "-type", "d", "-exec", "chmod", "a+rwx,a-t", "{}", "+",
		}, args)
		err := filepath.WalkDir(path, func(directory string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return os.Chmod(directory, 0o777)
			}
			return nil
		})
		return "", "", err
	}
	require.NoError(t, provisioner.Cleanup(context.Background(), workspace))
	assert.Equal(t, 1, calls)
	assert.Equal(t, 2, removals)
	assert.NoDirExists(t, path)
	assert.FileExists(t, filepath.Join(published, "artifact"))
	info, err := os.Stat(published)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm(), "published artifacts must not be chmodded through workspace symlinks")
	require.NoError(t, provisioner.Cleanup(context.Background(), workspace))
	assert.Equal(t, 1, calls, "normal or already completed cleanup must not launch a helper")
}

func TestHostWorkspaceCleanupPermissionRepairFailuresAreActionable(t *testing.T) {
	root := t.TempDir()
	previous := viper.Get(common.V2RuntimeWorkspaceRoot)
	viper.Set(common.V2RuntimeWorkspaceRoot, root)
	t.Cleanup(func() { viper.Set(common.V2RuntimeWorkspaceRoot, previous) })
	provisioner := hostPathWorkspaceProvisioner{}
	workspace, err := provisioner.Provision(context.Background(), WorkspaceRequest{RunID: "run"})
	require.NoError(t, err)
	calls := 0
	failure := errors.New("Docker daemon unavailable")
	provisioner.command = func(context.Context, ...string) (string, string, error) {
		calls++
		return "", "helper image unavailable", failure
	}
	provisioner.removeAll = func(string) error { return errors.New("filesystem I/O error") }
	require.ErrorContains(t, provisioner.Cleanup(context.Background(), workspace), "filesystem I/O error")
	assert.Zero(t, calls, "non-permission failures must not start a privileged helper")
	provisioner.removeAll = func(string) error {
		return &os.PathError{Op: "unlinkat", Path: workspace.Metadata["path"], Err: os.ErrPermission}
	}
	err = provisioner.Cleanup(context.Background(), workspace)
	require.ErrorIs(t, err, failure)
	assert.Contains(t, err.Error(), workspace.Metadata["path"])
	assert.Contains(t, err.Error(), "alpine:3.23")
	assert.Contains(t, err.Error(), "ensure Docker is available and the helper image is loaded")
	assert.Equal(t, 1, calls)
	workspace.Metadata["path"] = root
	require.ErrorContains(t, provisioner.Cleanup(context.Background(), workspace), "derived")
	assert.Equal(t, 1, calls, "a forged workspace must never reach the helper")
}

func TestPVCWorkspaceRequiresClientAndNamespace(t *testing.T) {
	_, err := (pvcWorkspaceProvisioner{}).Provision(context.Background(), WorkspaceRequest{RunID: "run", RunName: "name", Namespace: "ns"})
	require.ErrorContains(t, err, "client")
	_, err = (pvcWorkspaceProvisioner{coreClient: &recordingCoreClient{clientSet: fake.NewSimpleClientset()}}).Provision(
		context.Background(), WorkspaceRequest{RunID: "run", RunName: "name"})
	require.ErrorContains(t, err, "namespace")
}

func TestPVCWorkspaceIdempotentCreateRaceAndIdentity(t *testing.T) {
	client := fake.NewSimpleClientset()
	request := WorkspaceRequest{RunID: "run", RunName: "run-name", Namespace: "ns", Requirements: &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
	}}
	name := driver.GetWorkspacePVCName(request.RunName)
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: "ns", UID: "pvc-uid", Labels: map[string]string{attemptRunLabel: "run"},
	}, Spec: *request.Requirements}
	getCount := 0
	client.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getCount++
		if getCount == 1 {
			return true, nil, apierrors.NewNotFound(schema.GroupResource{Resource: "persistentvolumeclaims"}, name)
		}
		return false, nil, nil
	})
	client.PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		require.Equal(t, request.Requirements.AccessModes, action.(k8stesting.CreateAction).GetObject().(*corev1.PersistentVolumeClaim).Spec.AccessModes)
		require.NoError(t, client.Tracker().Add(pvc))
		return true, nil, apierrors.NewAlreadyExists(schema.GroupResource{Resource: "persistentvolumeclaims"}, name)
	})
	provisioner := pvcWorkspaceProvisioner{coreClient: &recordingCoreClient{clientSet: client}}
	workspace, err := provisioner.Provision(context.Background(), request)
	require.NoError(t, err)
	same, err := provisioner.Provision(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, workspace, same)
	_, err = provisioner.ResolveAttachment(context.Background(), workspace, ExecutionTarget{Executor: ExecutorKubernetes, Namespace: "ns"})
	require.NoError(t, err)
	_, err = provisioner.ResolveAttachment(context.Background(), workspace, ExecutionTarget{Executor: ExecutorKubernetes, Namespace: "other"})
	require.Error(t, err)
	client.PrependReactor("delete", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		options := action.(k8stesting.DeleteAction).GetDeleteOptions()
		require.NotNil(t, options.Preconditions)
		assert.Equal(t, pvc.UID, *options.Preconditions.UID)
		return false, nil, nil
	})
	require.NoError(t, provisioner.Cleanup(context.Background(), workspace))
	require.NoError(t, provisioner.Cleanup(context.Background(), workspace))
	pvc.UID = "replacement"
	require.NoError(t, client.Tracker().Add(pvc))
	require.ErrorContains(t, provisioner.Cleanup(context.Background(), workspace), "identity")
	request.RunID = "another"
	_, err = provisioner.Provision(context.Background(), request)
	require.ErrorContains(t, err, "another run")
}
