package poc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	apiserverclient "github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	attemptRunLabel  = "pipelines.kubeflow.org/run_id"
	attemptTaskLabel = "pipelines.kubeflow.org/task_id"
	attemptIDLabel   = "pipelines.kubeflow.org/attempt_id"
)

type kubernetesExecutor struct {
	coreClient apiserverclient.KubernetesCoreInterface
	namespace  string
}

var (
	_ WorkloadExecutorPlugin = kubernetesExecutor{}
	_ LostTaskRecoverer      = kubernetesExecutor{}
)

type podExecutionData struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	PodUID    string `json:"podUID"`
}

func (kubernetesExecutor) Type() string { return ExecutorKubernetes }
func (kubernetesExecutor) Capabilities() ExecutorCapabilities {
	return ExecutorCapabilities{RecoverLostTasks: true, SupportsMultiUser: true, SupportedWorkloadKinds: []WorkloadKind{WorkloadKindContainer}}
}
func (kubernetesExecutor) Capacity() ExecutorCapacity { return ExecutorCapacity{} }
func (k kubernetesExecutor) CheckRequirements(context.Context) error {
	if k.coreClient == nil {
		return fmt.Errorf("kubernetes executor requires a kubernetes core client")
	}
	return nil
}
func (k kubernetesExecutor) Start(ctx context.Context) error { return k.CheckRequirements(ctx) }
func (kubernetesExecutor) Shutdown(context.Context) error    { return nil }

func validateAttempt(identity TaskAttemptIdentity) error {
	if identity.RunID == "" || identity.TaskID == "" || identity.AttemptID == "" {
		return fmt.Errorf("run, task and attempt IDs are required")
	}
	return nil
}

func attemptWorkloadName(identity TaskAttemptIdentity) string {
	digest := sha256.Sum256([]byte(identity.RunID + "\x00" + identity.TaskID + "\x00" + identity.AttemptID))
	return fmt.Sprintf("kfp-task-%x", digest[:16])
}

func attemptLabels(identity TaskAttemptIdentity) map[string]string {
	return map[string]string{
		attemptRunLabel:  identity.RunID,
		attemptTaskLabel: identity.TaskID,
		attemptIDLabel:   identity.AttemptID,
	}
}

func checkAttemptLabels(labels map[string]string, identity TaskAttemptIdentity) error {
	for key, expected := range attemptLabels(identity) {
		if labels[key] != expected {
			return fmt.Errorf("workload identity mismatch for %s; refusing to adopt or stop it", key)
		}
	}
	return nil
}

func (k kubernetesExecutor) Submit(ctx context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	if err := k.CheckRequirements(ctx); err != nil {
		return ExecutionReference{}, err
	}
	if err := validateAttempt(request.Identity); err != nil {
		return ExecutionReference{}, err
	}
	if request.Namespace == "" || request.Pod == nil {
		return ExecutionReference{}, fmt.Errorf("kubernetes executor requires a namespace and a framework-prepared pod")
	}
	if request.Identity.Namespace != "" && request.Identity.Namespace != request.Namespace {
		return ExecutionReference{}, fmt.Errorf("attempt namespace differs from execution namespace")
	}
	pod := request.Pod.DeepCopy()
	if pod.Namespace != "" && pod.Namespace != request.Namespace {
		return ExecutionReference{}, fmt.Errorf("prepared pod namespace differs from execution namespace")
	}
	pod.Name, pod.GenerateName, pod.Namespace = attemptWorkloadName(request.Identity), "", request.Namespace
	pod.UID, pod.ResourceVersion = "", ""
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	for key, value := range attemptLabels(request.Identity) {
		pod.Labels[key] = value
	}
	pod.Labels[commonutil.LabelKeyWorkflowRunId] = request.Identity.RunID
	client := k.coreClient.PodClient(request.Namespace)
	created, err := client.Create(ctx, pod, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		created, err = client.Get(ctx, pod.Name, metav1.GetOptions{})
	}
	if err != nil {
		return ExecutionReference{}, err
	}
	if created == nil || created.UID == "" {
		return ExecutionReference{}, fmt.Errorf("created pod has no UID; cannot persist a safe execution reference")
	}
	if created.Name != pod.Name || created.Namespace != request.Namespace {
		return ExecutionReference{}, fmt.Errorf("created pod location differs from the requested attempt")
	}
	if request.Pod.UID != "" && created.UID != request.Pod.UID {
		return ExecutionReference{}, fmt.Errorf("existing pod UID differs from the prepared pod identity")
	}
	if err := checkAttemptLabels(created.Labels, request.Identity); err != nil {
		return ExecutionReference{}, err
	}
	return podReference(request.Identity, created), nil
}

func podReference(identity TaskAttemptIdentity, pod *corev1.Pod) ExecutionReference {
	return ExecutionReference{Executor: ExecutorKubernetes, Version: 1, Identity: identity, Data: model.JSONData{
		"namespace": pod.Namespace, "podName": pod.Name, "podUID": string(pod.UID),
	}}
}

func parsePodReference(reference ExecutionReference) (podExecutionData, error) {
	data := podExecutionData{}
	if reference.Executor != ExecutorKubernetes || reference.Version != 1 {
		return data, fmt.Errorf("unsupported kubernetes execution reference")
	}
	if err := validateAttempt(reference.Identity); err != nil {
		return data, err
	}
	raw, err := json.Marshal(reference.Data)
	if err == nil {
		err = json.Unmarshal(raw, &data)
	}
	if err != nil {
		return data, err
	}
	if data.Namespace == "" || data.PodName == "" || data.PodUID == "" || data.PodName != attemptWorkloadName(reference.Identity) {
		return data, fmt.Errorf("kubernetes execution reference is incomplete or has an inconsistent pod name")
	}
	if reference.Identity.Namespace != "" && data.Namespace != reference.Identity.Namespace {
		return data, fmt.Errorf("execution reference namespace differs from attempt namespace")
	}
	return data, nil
}

func checkReferencedPod(pod *corev1.Pod, reference ExecutionReference, data podExecutionData) error {
	if pod == nil || pod.Name != data.PodName || pod.Namespace != data.Namespace || string(pod.UID) != data.PodUID {
		return fmt.Errorf("pod UID or location changed; refusing to adopt or stop a replacement")
	}
	return checkAttemptLabels(pod.Labels, reference.Identity)
}

func podTaskResult(pod *corev1.Pod) (TaskExecutionResult, bool) {
	result := TaskExecutionResult{StatusMetadata: customStatusMetadata("", map[string]any{
		"executor": ExecutorKubernetes, "podName": pod.Name, "podUID": string(pod.UID), "phase": string(pod.Status.Phase),
	})}
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		result.Outcome = TaskOutcomeSuccess
	case corev1.PodFailed:
		result.Outcome = TaskOutcomeFailure
		result.Failure = &TaskFailure{Class: "workload", Message: pod.Status.Message}
		if pod.Status.Reason == "Evicted" || pod.Status.Reason == "NodeLost" {
			result.Outcome = TaskOutcomeError
			result.Failure.Class, result.Failure.Transient = "infrastructure", true
		}
	default:
		return TaskExecutionResult{}, false
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Terminated == nil {
			continue
		}
		if len(pod.Spec.Containers) != 0 && status.Name != pod.Spec.Containers[0].Name {
			continue
		}
		report, err := component.ParseRuntimeLauncherReport(status.State.Terminated.Message)
		if err != nil {
			continue
		}
		result.ComputationCompleted = report.ComputationCompleted
		result.PublicationCompleted = report.PublicationCompleted
		properties := result.StatusMetadata["customProperties"].(map[string]any)
		properties["launcherReport"] = report
		if report.AttemptID != pod.Labels[attemptIDLabel] {
			result.Outcome = TaskOutcomeError
			result.Failure = &TaskFailure{Class: "publication", Message: "launcher report belongs to a different attempt"}
			return result, true
		}
		if report.PublicationCompleted && pod.Status.Phase == corev1.PodSucceeded {
			result.Outcome, result.Failure = TaskOutcomeSuccess, nil
			return result, true
		}
		class := "publication"
		if report.Phase == component.RuntimePhaseOutputContractFailed {
			class = "output-contract"
		}
		message := report.Message
		if message == "" {
			message = "launcher stopped after computation but before publication completed"
		}
		result.Outcome = TaskOutcomeError
		result.Failure = &TaskFailure{Class: class, Message: message, Transient: report.Phase == component.RuntimePhasePublishing}
		return result, true
	}
	return result, true
}

func waitForPluginPoll(ctx context.Context) error {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (k kubernetesExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	if err := k.CheckRequirements(ctx); err != nil {
		return TaskExecutionResult{}, err
	}
	data, err := parsePodReference(reference)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	for {
		if err := ctx.Err(); err != nil {
			return TaskExecutionResult{}, err
		}
		pod, err := k.coreClient.PodClient(data.Namespace).Get(ctx, data.PodName, metav1.GetOptions{})
		if err != nil {
			return TaskExecutionResult{}, err
		}
		if err := checkReferencedPod(pod, reference, data); err != nil {
			return TaskExecutionResult{}, err
		}
		if result, terminal := podTaskResult(pod); terminal {
			return result, nil
		}
		if err := waitForPluginPoll(ctx); err != nil {
			return TaskExecutionResult{}, err
		}
	}
}

func (k kubernetesExecutor) Cancel(ctx context.Context, reference ExecutionReference) error {
	if err := k.CheckRequirements(ctx); err != nil {
		return err
	}
	data, err := parsePodReference(reference)
	if err != nil {
		return err
	}
	client := k.coreClient.PodClient(data.Namespace)
	pod, err := client.Get(ctx, data.PodName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := checkReferencedPod(pod, reference, data); err != nil {
		return err
	}
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return nil
	}
	if pod.Spec.NodeName == "" && pod.Status.StartTime == nil {
		uid := types.UID(data.PodUID)
		err = client.Delete(ctx, data.PodName, metav1.DeleteOptions{
			Preconditions: &metav1.Preconditions{UID: &uid, ResourceVersion: &pod.ResourceVersion},
		})
	} else if pod.Spec.ActiveDeadlineSeconds == nil || *pod.Spec.ActiveDeadlineSeconds > 1 {
		// Stop the workload without deleting its log source. Another replica
		// must still be able to publish canceled-attempt logs after takeover.
		deadline := int64(1)
		pod.Spec.ActiveDeadlineSeconds = &deadline
		_, err = client.Update(ctx, pod, metav1.UpdateOptions{})
	}
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for {
		pod, err := client.Get(ctx, data.PodName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if pod != nil && string(pod.UID) != data.PodUID {
			return nil
		}
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return nil
		}
		if err := waitForPluginPoll(ctx); err != nil {
			return err
		}
	}
}

func (k kubernetesExecutor) Recover(ctx context.Context, identity TaskAttemptIdentity, reference *ExecutionReference) (RecoveryResult, error) {
	unknown := RecoveryResult{State: RecoveryUnknown}
	if err := k.CheckRequirements(ctx); err != nil {
		return unknown, err
	}
	if err := validateAttempt(identity); err != nil {
		return unknown, err
	}
	namespace := identity.Namespace
	if namespace == "" {
		namespace = k.namespace
	}
	data := podExecutionData{Namespace: namespace, PodName: attemptWorkloadName(identity)}
	if reference != nil {
		if reference.Identity != identity {
			return unknown, fmt.Errorf("recovery identity differs from execution reference")
		}
		var err error
		data, err = parsePodReference(*reference)
		if err != nil {
			return unknown, err
		}
	}
	if data.Namespace == "" {
		return unknown, fmt.Errorf("namespace is required to recover a pod without a reference")
	}
	pod, err := k.coreClient.PodClient(data.Namespace).Get(ctx, data.PodName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return RecoveryResult{State: RecoveryStopped}, nil
	}
	if err != nil {
		return unknown, err
	}
	if pod == nil || pod.UID == "" {
		return unknown, fmt.Errorf("pod lookup returned no durable identity")
	}
	if pod.Name != data.PodName || pod.Namespace != data.Namespace {
		return unknown, fmt.Errorf("recovered pod location differs from the requested attempt")
	}
	if err := checkAttemptLabels(pod.Labels, identity); err != nil {
		return unknown, err
	}
	if reference != nil {
		if err := checkReferencedPod(pod, *reference, data); err != nil {
			return unknown, err
		}
	}
	found := podReference(identity, pod)
	if result, terminal := podTaskResult(pod); terminal {
		return RecoveryResult{State: RecoveryCompleted, Reference: &found, Result: &result}, nil
	}
	return RecoveryResult{State: RecoveryExisting, Reference: &found}, nil
}
