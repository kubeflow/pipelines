package poc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const platformPVCRequestAnnotation = "pipelines.kubeflow.org/runtime-pvc-request"

func (c *Coordinator) createOrRecoverPlatformPVC(ctx context.Context, run *model.Run, task *model.Task, pvc *corev1.PersistentVolumeClaim) error {
	ctx, cancel, err := c.ownership.operation(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	spec, err := json.Marshal(pvc.Spec)
	if err != nil {
		return fmt.Errorf("encode PVC task requirements: %w", err)
	}
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(spec))
	pvc.Labels = map[string]string{attemptRunLabel: run.UUID, attemptTaskLabel: task.UUID}
	pvc.Annotations[platformPVCRequestAnnotation] = fingerprint
	client := c.kubernetesCoreClient.GetClientSet().CoreV1().PersistentVolumeClaims(pvc.Namespace)
	_, err = client.Create(ctx, pvc, metav1.CreateOptions{})
	if !apierrors.IsAlreadyExists(err) {
		return err
	}
	existing, err := client.Get(ctx, pvc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if existing.Labels[attemptRunLabel] != run.UUID || existing.Labels[attemptTaskLabel] != task.UUID ||
		existing.Annotations[platformPVCRequestAnnotation] != fingerprint {
		return fmt.Errorf("existing PVC %s does not match the task identity and requirements", pvc.Name)
	}
	return nil
}
