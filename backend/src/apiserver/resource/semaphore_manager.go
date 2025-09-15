package resource

import (
	"context"
	"os"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	v1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Common constants for semaphore functionality
const (
	SemaphoreConfigName = "semaphore-config"
)

// Common Kubernetes labels used across KFP resources
const (
	LabelApp       = "kubeflow-pipelines"
	LabelManagedBy = "kubeflow-pipelines-apiserver"
)

const (
	LabelKeyApp       = "app"
	LabelKeyComponent = "component"
	LabelKeyManagedBy = "app.kubernetes.io/managed-by"
)

func createKFPLabels(component string) map[string]string {
	return map[string]string{
		LabelKeyApp:       LabelApp,
		LabelKeyComponent: component,
		LabelKeyManagedBy: LabelManagedBy,
	}
}

type SemaphoreManager struct {
	k8sClient client.KubernetesCoreInterface
}

func NewSemaphoreManager(k8sClient client.KubernetesCoreInterface) *SemaphoreManager {
	return &SemaphoreManager{k8sClient: k8sClient}
}

// EnsureSemaphoreConfigMap creates a ConfigMap entry for the given semaphore key with default limit of 1.
// Only creates new entries - preserves existing user edits.
func (s *SemaphoreManager) EnsureSemaphoreConfigMap(ctx context.Context, namespace, semaphoreKey string) error {
	if semaphoreKey == "" {
		return nil
	}

	configMapName := s.getSemaphoreConfigMapName()
	configMapClient := s.k8sClient.ConfigMapClient(namespace)

	// Get existing ConfigMap
	configMap, err := configMapClient.Get(ctx, configMapName, metav1.GetOptions{})
	if k8errors.IsNotFound(err) {
		// Create new ConfigMap
		configMap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
				Labels:    createKFPLabels(SemaphoreConfigName),
			},
			Data: map[string]string{
				semaphoreKey: "1", // Default limit of 1 (mutex behavior)
			},
		}
		_, err = configMapClient.Create(ctx, configMap, metav1.CreateOptions{})
		if err != nil {
			return util.NewInternalServerError(err, "Failed to create semaphore ConfigMap")
		}
		glog.Infof("Created semaphore ConfigMap %s/%s with key %s=1", namespace, configMapName, semaphoreKey)
		return nil
	} else if err != nil {
		return util.NewInternalServerError(err, "Failed to get semaphore ConfigMap")
	}

	// ConfigMap exists - only add missing keys, preserve user edits
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	// Only add the key if it doesn't exist (preserves user-modified limits)
	if _, exists := configMap.Data[semaphoreKey]; !exists {
		configMap.Data[semaphoreKey] = "1" // Default limit of 1
		_, err = configMapClient.Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return util.NewInternalServerError(err, "Failed to update semaphore ConfigMap")
		}
		glog.Infof("Added semaphore key %s=1 to existing ConfigMap %s/%s", semaphoreKey, namespace, configMapName)
	}

	return nil
}

// getSemaphoreConfigMapName returns the name of the ConfigMap used for semaphores
func (s *SemaphoreManager) getSemaphoreConfigMapName() string {
	if name := os.Getenv("SEMAPHORE_CONFIGMAP_NAME"); name != "" {
		return name
	}
	return SemaphoreConfigName
}
