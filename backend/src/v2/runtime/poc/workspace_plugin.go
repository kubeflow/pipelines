package poc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	apiserverclient "github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type hostPathWorkspaceProvisioner struct {
	command   dockerCommand
	removeAll func(string) error
}

// Offline Docker workers must preload this image for container-owned workspace cleanup.
const workspaceCleanupImage = "alpine:3.23"

var (
	_ WorkspaceProvisionerPlugin = hostPathWorkspaceProvisioner{}
	_ WorkspaceProvisionerPlugin = pvcWorkspaceProvisioner{}
)

func (hostPathWorkspaceProvisioner) Type() string { return WorkspaceTypeHostPath }

func hostWorkspacePath(runID string) (string, error) {
	if runID == "" || runID == "." || runID == ".." || strings.ContainsAny(runID, `/\`) {
		return "", fmt.Errorf("a single safe run ID is required for a workspace")
	}
	root, err := filepath.Abs(WorkspaceRoot())
	if err != nil {
		return "", err
	}
	return filepath.Join(root, runID, "workspace"), nil
}

func checkWorkspaceSymlinks(path string) error {
	for _, candidate := range []string{filepath.Dir(path), path} {
		info, err := os.Lstat(candidate)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("workspace path %q is a symlink; refusing unsafe access", candidate)
		}
	}
	return nil
}

func (hostPathWorkspaceProvisioner) Provision(ctx context.Context, request WorkspaceRequest) (*WorkspaceHandle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := hostWorkspacePath(request.RunID)
	if err != nil {
		return nil, err
	}
	if err := checkWorkspaceSymlinks(path); err != nil {
		return nil, err
	}
	if err := ensureContainerWritableDir(path); err != nil {
		return nil, err
	}
	return &WorkspaceHandle{Version: 1, Type: WorkspaceTypeHostPath, Metadata: map[string]string{
		"runID": request.RunID, "path": path,
	}}, nil
}

func validateHostWorkspace(workspace *WorkspaceHandle) (string, error) {
	if workspace == nil || workspace.Version != 1 || workspace.Type != WorkspaceTypeHostPath {
		return "", fmt.Errorf("a version 1 host-path workspace handle is required")
	}
	path, err := hostWorkspacePath(workspace.Metadata["runID"])
	if err != nil {
		return "", err
	}
	if workspace.Metadata["path"] != path {
		return "", fmt.Errorf("workspace path differs from its derived run workspace; refusing unsafe access")
	}
	if err := checkWorkspaceSymlinks(path); err != nil {
		return "", err
	}
	return path, nil
}

func (hostPathWorkspaceProvisioner) ResolveAttachment(ctx context.Context, workspace *WorkspaceHandle, target ExecutionTarget) (*WorkspaceHandle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if target.Executor != ExecutorDocker {
		return nil, fmt.Errorf("host-path workspace supports only Docker execution")
	}
	path, err := validateHostWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace path is not a directory")
	}
	return workspace, nil
}

func (p hostPathWorkspaceProvisioner) Cleanup(ctx context.Context, workspace *WorkspaceHandle) error {
	if workspace == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := validateHostWorkspace(workspace)
	if err != nil {
		return err
	}
	removeAll := p.removeAll
	if removeAll == nil {
		removeAll = os.RemoveAll
	}
	err = removeAll(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("cannot remove run workspace %q: %w", path, err)
	}
	if _, err := validateHostWorkspace(workspace); err != nil {
		return err
	}
	if err := p.repairWorkspacePermissions(ctx, path); err != nil {
		return err
	}
	if _, err := validateHostWorkspace(workspace); err != nil {
		return err
	}
	if err := removeAll(path); err != nil {
		return fmt.Errorf("permission repair completed but cannot remove run workspace %q: %w", path, err)
	}
	return nil
}

func (p hostPathWorkspaceProvisioner) repairWorkspacePermissions(ctx context.Context, path string) error {
	if strings.ContainsAny(path, ",:\r\n\"") {
		return fmt.Errorf("workspace path %q cannot be safely represented as a Docker bind mount", path)
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	args := []string{
		"run", "--rm", "--network", "none", "--read-only", "--user", "0:0",
		"--cap-drop", "ALL", "--cap-add", "FOWNER", "--cap-add", "DAC_OVERRIDE",
		"--security-opt", "no-new-privileges", "--pids-limit", "16", "--memory", "64m", "--cpus", "0.25",
		"--volume", bindMount(path, "/workspace"),
		workspaceCleanupImage, "timeout", "-s", "KILL", "30",
		// Physical traversal never follows workspace symlinks to published artifacts.
		"find", "/workspace", "-xdev", "-type", "d", "-exec", "chmod", "a+rwx,a-t", "{}", "+",
	}
	_, stderr, err := (dockerExecutor{command: p.command}).run(ctx, args...)
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return fmt.Errorf("cannot repair container-owned run workspace %q with %s: %w: %s; ensure Docker is available and the helper image is loaded",
			path, workspaceCleanupImage, err, strings.TrimSpace(stderr))
	}
	return nil
}

type pvcWorkspaceProvisioner struct {
	coreClient           apiserverclient.KubernetesCoreInterface
	defaultWorkspaceSpec *corev1.PersistentVolumeClaimSpec
}

func (pvcWorkspaceProvisioner) Type() string { return WorkspaceTypePVC }

func (p pvcWorkspaceProvisioner) checkClient(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("workspace PVC requires an explicit namespace")
	}
	if p.coreClient == nil || p.coreClient.GetClientSet() == nil {
		return fmt.Errorf("workspace PVC requires a Kubernetes client")
	}
	return nil
}

func (p pvcWorkspaceProvisioner) Provision(ctx context.Context, request WorkspaceRequest) (*WorkspaceHandle, error) {
	if err := p.checkClient(request.Namespace); err != nil {
		return nil, err
	}
	if request.RunID == "" || request.RunName == "" {
		return nil, fmt.Errorf("run ID and run name are required for a workspace PVC")
	}
	name := driver.GetWorkspacePVCName(request.RunName)
	client := p.coreClient.GetClientSet().CoreV1().PersistentVolumeClaims(request.Namespace)
	pvc, err := client.Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		spec := request.Requirements
		if spec == nil {
			spec = p.defaultWorkspaceSpec
		}
		if spec == nil {
			spec = &corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				}},
			}
		}
		pvc, err = client.Create(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: request.Namespace, Labels: map[string]string{attemptRunLabel: request.RunID}},
			Spec:       *spec.DeepCopy(),
		}, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			pvc, err = client.Get(ctx, name, metav1.GetOptions{})
		}
	}
	if err != nil {
		return nil, fmt.Errorf("cannot provision workspace PVC: %w", err)
	}
	return workspacePVCReference(pvc, request)
}

func workspacePVCReference(pvc *corev1.PersistentVolumeClaim, request WorkspaceRequest) (*WorkspaceHandle, error) {
	if pvc == nil || pvc.UID == "" || pvc.Labels[attemptRunLabel] != request.RunID ||
		pvc.Namespace != request.Namespace || pvc.Name != driver.GetWorkspacePVCName(request.RunName) {
		return nil, fmt.Errorf("workspace PVC has no UID or belongs to another run")
	}
	return &WorkspaceHandle{Version: 1, Type: WorkspaceTypePVC, Metadata: map[string]string{
		"runID": request.RunID, "namespace": request.Namespace, "pvcName": pvc.Name, "pvcUID": string(pvc.UID),
	}}, nil
}

func (p pvcWorkspaceProvisioner) recoverForCleanup(ctx context.Context, request WorkspaceRequest) (*WorkspaceHandle, error) {
	if err := p.checkClient(request.Namespace); err != nil {
		return nil, err
	}
	pvc, err := p.coreClient.GetClientSet().CoreV1().PersistentVolumeClaims(request.Namespace).
		Get(ctx, driver.GetWorkspacePVCName(request.RunName), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return workspacePVCReference(pvc, request)
}

func (p pvcWorkspaceProvisioner) validateHandle(workspace *WorkspaceHandle) error {
	if workspace == nil || workspace.Version != 1 || workspace.Type != WorkspaceTypePVC ||
		workspace.Metadata["pvcName"] == "" || workspace.Metadata["pvcUID"] == "" || workspace.Metadata["runID"] == "" {
		return fmt.Errorf("a complete version 1 PVC workspace handle is required")
	}
	return p.checkClient(workspace.Metadata["namespace"])
}

func checkWorkspacePVC(pvc *corev1.PersistentVolumeClaim, workspace *WorkspaceHandle) error {
	if pvc == nil || string(pvc.UID) != workspace.Metadata["pvcUID"] || pvc.Labels[attemptRunLabel] != workspace.Metadata["runID"] {
		return fmt.Errorf("workspace PVC identity changed; refusing to attach or delete a replacement")
	}
	return nil
}

func (p pvcWorkspaceProvisioner) ResolveAttachment(ctx context.Context, workspace *WorkspaceHandle, target ExecutionTarget) (*WorkspaceHandle, error) {
	if err := p.validateHandle(workspace); err != nil {
		return nil, err
	}
	if target.Executor != ExecutorKubernetes || target.Namespace != workspace.Metadata["namespace"] {
		return nil, fmt.Errorf("PVC workspace requires Kubernetes execution in its own namespace")
	}
	pvc, err := p.coreClient.GetClientSet().CoreV1().PersistentVolumeClaims(target.Namespace).Get(ctx, workspace.Metadata["pvcName"], metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if err := checkWorkspacePVC(pvc, workspace); err != nil {
		return nil, err
	}
	return workspace, nil
}

func (p pvcWorkspaceProvisioner) Cleanup(ctx context.Context, workspace *WorkspaceHandle) error {
	if workspace == nil {
		return nil
	}
	if err := p.validateHandle(workspace); err != nil {
		return err
	}
	client := p.coreClient.GetClientSet().CoreV1().PersistentVolumeClaims(workspace.Metadata["namespace"])
	pvc, err := client.Get(ctx, workspace.Metadata["pvcName"], metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := checkWorkspacePVC(pvc, workspace); err != nil {
		return err
	}
	uid := types.UID(workspace.Metadata["pvcUID"])
	err = client.Delete(ctx, pvc.Name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
