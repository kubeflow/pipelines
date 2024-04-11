// Copyright 2020 kubeflow.org
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	exec "github.com/kubeflow/pipelines/backend/src/common"
	swfregister "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	prclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	prclientv1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1"
	prsinformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	prinformer "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/cache"
)

// PipelineRun is a type to help manipulate PipelineRun objects.
type PipelineRun struct {
	*pipelineapi.PipelineRun
	// +optional
	Status TektonStatus `json:"status,omitempty"`
}

type TektonStatus struct {
	*pipelineapi.PipelineRunStatus
	// +optional
	TaskRuns map[string]*pipelineapi.PipelineRunTaskRunStatus `json:"taskRuns,omitempty"`
	// +optional
	Runs map[string]*pipelineapi.PipelineRunRunStatus `json:"runs,omitempty"`
}

type runKinds []string

var (
	// A list of Kinds that contains childReferences
	// those childReferences would be scaned and retrieve their taskrun/run status
	childReferencesKinds runKinds = []string{}
)

const (
	childReferencesKindFlagName = "childReferencesKinds"
)

func (rk *runKinds) String() string {
	return fmt.Sprint(*rk)
}

func (rk *runKinds) Set(value string) error {
	if len(*rk) > 0 {
		return fmt.Errorf("%s has been set", childReferencesKindFlagName)
	}

	for _, k := range strings.Split(value, ",") {
		*rk = append(*rk, k)
	}
	sort.Strings(*rk)

	return nil
}

func init() {
	flag.Var(&childReferencesKinds, childReferencesKindFlagName, "A list of kinds to search for the nested childReferences")
}

func NewPipelineRunFromBytes(bytes []byte) (*PipelineRun, error) {
	var pr pipelineapi.PipelineRun
	err := yaml.Unmarshal(bytes, &pr)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to unmarshal the inputs")
	}
	return NewPipelineRun(&pr), nil
}

func NewPipelineRunFromBytesJSON(bytes []byte) (*PipelineRun, error) {
	var pr pipelineapi.PipelineRun
	err := json.Unmarshal(bytes, &pr)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to unmarshal the inputs")
	}
	return NewPipelineRun(&pr), nil
}

func NewPipelineRunFromInterface(obj interface{}) (*PipelineRun, error) {
	pr, ok := obj.(*pipelineapi.PipelineRun)
	if ok {
		return NewPipelineRun(pr), nil
	}
	return nil, NewInvalidInputError("not PipelineRun struct")
}

func UnmarshParametersPipelineRun(paramsString string) (SpecParameters, error) {
	if paramsString == "" {
		return nil, nil
	}
	var params []pipelineapi.Param
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		return nil, NewInternalServerError(err, "Parameters have wrong format")
	}
	rev := make(SpecParameters, 0, len(params))
	for _, param := range params {
		rev = append(rev, SpecParameter{
			Name:  param.Name,
			Value: StringPointer(param.Value.StringVal)})
	}
	return rev, nil
}

func MarshalParametersPipelineRun(params SpecParameters) (string, error) {
	if params == nil {
		return "[]", nil
	}

	inputParams := make([]pipelineapi.Param, 0)
	for _, param := range params {
		newParam := pipelineapi.Param{
			Name:  param.Name,
			Value: pipelineapi.ParamValue{Type: "string", StringVal: *param.Value},
		}
		inputParams = append(inputParams, newParam)
	}
	paramBytes, err := json.Marshal(inputParams)
	if err != nil {
		return "", NewInvalidInputErrorWithDetails(err, "Failed to marshal the parameter.")
	}
	if len(paramBytes) > MaxParameterBytes {
		return "", NewInvalidInputError("The input parameter length exceed maximum size of %v.", MaxParameterBytes)
	}
	return string(paramBytes), nil
}

func NewPipelineRunFromScheduleWorkflowSpecBytesJSON(bytes []byte) (*PipelineRun, error) {
	var pr pipelineapi.PipelineRun
	err := json.Unmarshal(bytes, &pr.Spec)
	if err != nil {
		return nil, NewInvalidInputErrorWithDetails(err, "Failed to unmarshal the inputs")
	}
	pr.APIVersion = "tekton.dev/v1"
	pr.Kind = "PipelineRun"
	return NewPipelineRun(&pr), nil
}

// NewWorkflow creates a Workflow.
func NewPipelineRun(pr *pipelineapi.PipelineRun) *PipelineRun {
	return &PipelineRun{
		pr,
		TektonStatus{&pr.Status, map[string]*pipelineapi.PipelineRunTaskRunStatus{}, map[string]*pipelineapi.PipelineRunRunStatus{}},
	}
}

func (w *PipelineRun) GetWorkflowParametersAsMap() map[string]string {
	resultAsArray := w.Spec.Params
	resultAsMap := make(map[string]string)
	for _, param := range resultAsArray {
		resultAsMap[param.Name] = param.Value.StringVal
	}
	return resultAsMap
}

// SetServiceAccount Set the service account to run the workflow.
func (pr *PipelineRun) SetServiceAccount(serviceAccount string) {
	pr.Spec.TaskRunTemplate.ServiceAccountName = serviceAccount
}

// OverrideParameters overrides some of the parameters of a Workflow.
func (pr *PipelineRun) OverrideParameters(desiredParams map[string]string) {
	desiredSlice := make([]pipelineapi.Param, 0)
	for _, currentParam := range pr.Spec.Params {
		var desiredValue pipelineapi.ParamValue = pipelineapi.ParamValue{
			Type:      "string",
			StringVal: "",
		}
		if param, ok := desiredParams[currentParam.Name]; ok {
			desiredValue.StringVal = param
		} else {
			desiredValue.StringVal = currentParam.Value.StringVal
		}
		desiredSlice = append(desiredSlice, pipelineapi.Param{
			Name:  currentParam.Name,
			Value: desiredValue,
		})
	}
	pr.Spec.Params = desiredSlice
}

func (pr *PipelineRun) VerifyParameters(desiredParams map[string]string) error {
	templateParamsMap := make(map[string]*string)
	for _, param := range pr.Spec.Params {
		templateParamsMap[param.Name] = &param.Value.StringVal
	}
	for k := range desiredParams {
		_, ok := templateParamsMap[k]
		if !ok {
			glog.Warningf("Unrecognized input parameter: %v", k)
		}
	}
	return nil
}

func (pr *PipelineRun) ScheduledWorkflowUUIDAsStringOrEmpty() string {
	if pr.OwnerReferences == nil {
		return ""
	}

	for _, reference := range pr.OwnerReferences {
		if isScheduledWorkflow(reference) {
			return string(reference.UID)
		}
	}

	return ""
}

func (pr *PipelineRun) ScheduledAtInSecOr0() int64 {
	if pr.Labels == nil {
		return 0
	}

	for key, value := range pr.Labels {
		if key == LabelKeyWorkflowEpoch {
			result, err := RetrieveInt64FromLabel(value)
			if err != nil {
				glog.Errorf("Could not retrieve scheduled epoch from label key (%v) and label value (%v).", key, value)
				return 0
			}
			return result
		}
	}

	return 0
}

func (pr *PipelineRun) FinishedAt() int64 {
	if pr.Status.PipelineRunStatusFields.CompletionTime.IsZero() {
		// If workflow is not finished
		return 0
	}
	return pr.Status.PipelineRunStatusFields.CompletionTime.Unix()
}

func (pr *PipelineRun) FinishedAtTime() metav1.Time {
	return *pr.Status.PipelineRunStatusFields.CompletionTime
}

func (pr *PipelineRun) Condition() exec.ExecutionPhase {
	if len(pr.Status.Conditions) > 0 {
		switch pr.Status.Conditions[0].Reason {
		case "Error":
			return exec.ExecutionError
		case "Failed":
			return exec.ExecutionFailed
		case "InvalidTaskResultReference":
			return exec.ExecutionFailed
		case "Cancelled":
			return exec.ExecutionFailed
		case "Pending":
			return exec.ExecutionPending
		case "Running":
			return exec.ExecutionRunning
		case "Succeeded":
			return exec.ExecutionSucceeded
		case "Completed":
			return exec.ExecutionSucceeded
		case "PipelineRunTimeout":
			return exec.ExecutionError
		case "PipelineRunCancelled":
			return exec.ExecutionPhase("Terminated")
		case "PipelineRunCouldntCancel":
			return exec.ExecutionError
		case "Terminating":
			return exec.ExecutionPhase("Terminating")
		case "Terminated":
			return exec.ExecutionPhase("Terminated")
		default:
			return exec.ExecutionUnknown
		}
	} else {
		return exec.ExecutionUnknown
	}
}

func (pr *PipelineRun) ToStringForStore() string {
	workflow, err := json.Marshal(pr)
	if err != nil {
		glog.Errorf("Could not marshal the workflow: %v", pr)
		return ""
	}
	return string(workflow)
}

func (pr *PipelineRun) HasScheduledWorkflowAsParent() bool {
	return containsScheduledWorkflow(pr.PipelineRun.OwnerReferences)
}

func (pr *PipelineRun) GetExecutionSpec() ExecutionSpec {
	pipelinerun := pr.DeepCopy()
	pipelinerun.Status = pipelineapi.PipelineRunStatus{}
	pipelinerun.TypeMeta = metav1.TypeMeta{Kind: pr.Kind, APIVersion: pr.APIVersion}
	// To prevent collisions, clear name, set GenerateName to first 200 runes of previous name.
	nameRunes := []rune(pr.Name)
	length := len(nameRunes)
	if length > 200 {
		length = 200
	}
	pipelinerun.ObjectMeta = metav1.ObjectMeta{GenerateName: string(nameRunes[:length])}
	return NewPipelineRun(pipelinerun)
}

// OverrideName sets the name of a Workflow.
func (pr *PipelineRun) OverrideName(name string) {
	pr.GenerateName = ""
	pr.Name = name
}

// SetAnnotationsToAllTemplatesIfKeyNotExist sets annotations on all templates in a Workflow
// if the annotation key does not exist
func (pr *PipelineRun) SetAnnotationsToAllTemplatesIfKeyNotExist(key string, value string) {
	// No metadata object within pipelineRun task
}

// SetLabels sets labels on all templates in a Workflow
func (pr *PipelineRun) SetLabelsToAllTemplates(key string, value string) {
	// No metadata object within pipelineRun task
}

// SetOwnerReferences sets owner references on a Workflow.
func (pr *PipelineRun) SetOwnerReferences(schedule *swfapi.ScheduledWorkflow) {
	pr.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(schedule, schema.GroupVersionKind{
			Group:   swfapi.SchemeGroupVersion.Group,
			Version: swfapi.SchemeGroupVersion.Version,
			Kind:    swfregister.Kind,
		}),
	}
}

func (pr *PipelineRun) SetLabels(key string, value string) {
	if pr.Labels == nil {
		pr.Labels = make(map[string]string)
	}
	pr.Labels[key] = value
}

func (pr *PipelineRun) SetAnnotations(key string, value string) {
	if pr.Annotations == nil {
		pr.Annotations = make(map[string]string)
	}
	pr.Annotations[key] = value
}

func (pr *PipelineRun) ReplaceUID(id string) error {
	newWorkflowString := strings.Replace(pr.ToStringForStore(), "{{workflow.uid}}", id, -1)
	newWorkflowString = strings.Replace(newWorkflowString, "$(context.pipelineRun.uid)", id, -1)
	var workflow *pipelineapi.PipelineRun
	if err := json.Unmarshal([]byte(newWorkflowString), &workflow); err != nil {
		return NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow: %s", pr.ToStringForStore())
	}
	pr.PipelineRun = workflow
	return nil
}

func (pr *PipelineRun) ReplaceOrignalPipelineRunName(name string) error {
	newWorkflowString := strings.Replace(pr.ToStringForStore(), "$ORIG_PR_NAME", name, -1)
	var workflow *pipelineapi.PipelineRun
	if err := json.Unmarshal([]byte(newWorkflowString), &workflow); err != nil {
		return NewInternalServerError(err,
			"Failed to unmarshal workflow spec manifest. Workflow: %s", pr.ToStringForStore())
	}
	pr.PipelineRun = workflow
	return nil
}

func (pr *PipelineRun) SetCannonicalLabels(name string, nextScheduledEpoch int64, index int64) {
	pr.SetLabels(LabelKeyWorkflowScheduledWorkflowName, name)
	pr.SetLabels(LabelKeyWorkflowEpoch, FormatInt64ForLabel(nextScheduledEpoch))
	pr.SetLabels(LabelKeyWorkflowIndex, FormatInt64ForLabel(index))
	pr.SetLabels(LabelKeyWorkflowIsOwnedByScheduledWorkflow, "true")
}

// FindObjectStoreArtifactKeyOrEmpty loops through all node running statuses and look up the first
// S3 artifact with the specified nodeID and artifactName. Returns empty if nothing is found.
func (pr *PipelineRun) FindObjectStoreArtifactKeyOrEmpty(nodeID string, artifactName string) string {
	// TODO: The below artifact keys are only for parameter artifacts. Will need to also implement
	//       metric and raw input artifacts once we finallized the big data passing in our compiler.

	if pr.Status.TaskRuns == nil {
		return ""
	}
	return "artifacts/" + pr.ObjectMeta.Name + "/" + nodeID + "/" + artifactName + ".tgz"
}

// FindTaskRunByPodName loops through all workflow task runs and look up by the pod name.
func (pr *PipelineRun) FindTaskRunByPodName(podName string) (*pipelineapi.PipelineRunTaskRunStatus, string) {
	for id, taskRun := range pr.Status.TaskRuns {
		if taskRun.Status.PodName == podName {
			return taskRun, id
		}
	}
	return nil, ""
}

// IsInFinalState whether the workflow is in a final state.
func (pr *PipelineRun) IsInFinalState() bool {
	// Workflows in the statuses other than pending or running are considered final.

	if len(pr.Status.Conditions) > 0 {
		finalConditions := map[string]int{
			"Succeeded":                  1,
			"Failed":                     1,
			"Completed":                  1,
			"PipelineRunCancelled":       1, // remove this when Tekton move to v1 API
			"PipelineRunCouldntCancel":   1,
			"PipelineRunTimeout":         1,
			"Cancelled":                  1,
			"StoppedRunFinally":          1,
			"CancelledRunFinally":        1,
			"InvalidTaskResultReference": 1,
		}
		phase := pr.Status.Conditions[0].Reason
		if _, ok := finalConditions[phase]; ok {
			return true
		}
	}
	return false
}

// PersistedFinalState whether the workflow final state has being persisted.
func (pr *PipelineRun) PersistedFinalState() bool {
	if _, ok := pr.GetLabels()[LabelKeyWorkflowPersistedFinalState]; ok {
		// If the label exist, workflow final state has being persisted.
		return true
	}
	return false
}

// IsV2Compatible whether the workflow is a v2 compatible pipeline.
func (pr *PipelineRun) IsV2Compatible() bool {
	value := pr.GetObjectMeta().GetAnnotations()["pipelines.kubeflow.org/v2_pipeline"]
	return value == "true"
}

// no compression/decompression in tekton
func (pr *PipelineRun) Decompress() error {
	return nil
}

// Always can retry
func (pr *PipelineRun) CanRetry() error {
	return nil
}

func (pr *PipelineRun) ExecutionName() string {
	return pr.Name
}

func (pr *PipelineRun) SetExecutionName(name string) {
	pr.GenerateName = ""
	pr.Name = name

}

func (pr *PipelineRun) ExecutionNamespace() string {
	return pr.Namespace
}

func (pr *PipelineRun) SetExecutionNamespace(namespace string) {
	pr.Namespace = namespace
}

func (pr *PipelineRun) ExecutionObjectMeta() *metav1.ObjectMeta {
	return &pr.ObjectMeta
}

func (pr *PipelineRun) ExecutionTypeMeta() *metav1.TypeMeta {
	return &pr.TypeMeta
}

func (pr *PipelineRun) ExecutionStatus() ExecutionStatus {
	return pr
}

func (pr *PipelineRun) ExecutionType() ExecutionType {
	return TektonPipelineRun
}

func (pr *PipelineRun) ExecutionUID() string {
	return string(pr.UID)
}

func (pr *PipelineRun) HasMetrics() bool {
	return pr.Status.TaskRuns != nil && pr.Status.Runs != nil
}

func (pr *PipelineRun) Message() string {
	if pr.Status.Conditions != nil && len(pr.Status.Conditions) > 0 {
		return pr.Status.Conditions[0].Message
	}
	return ""
}

func (pr *PipelineRun) StartedAtTime() metav1.Time {
	return *pr.Status.PipelineRunStatusFields.StartTime
}

func (pr *PipelineRun) IsTerminating() bool {
	return pr.Spec.Status == "Cancelled" && !pr.IsDone()
}

func (pr *PipelineRun) ServiceAccount() string {
	return pr.Spec.TaskRunTemplate.ServiceAccountName
}

func (pr *PipelineRun) SetPodMetadataLabels(key string, value string) {
	if pr.Labels == nil {
		pr.Labels = make(map[string]string)
	}
	pr.Labels[key] = value
}

func (pr *PipelineRun) SetSpecParameters(params SpecParameters) {
	desiredSlice := make([]pipelineapi.Param, 0)
	for _, currentParam := range params {
		newParam := pipelineapi.Param{
			Name: currentParam.Name,
			Value: pipelineapi.ParamValue{
				Type:      "string",
				StringVal: *currentParam.Value,
			},
		}
		desiredSlice = append(desiredSlice, newParam)
	}
	pr.Spec.Params = desiredSlice
}

func (pr *PipelineRun) Version() string {
	return pr.ResourceVersion
}

func (pr *PipelineRun) SetVersion(version string) {
	pr.ResourceVersion = version
}

func (pr *PipelineRun) SpecParameters() SpecParameters {
	rev := make(SpecParameters, 0, len(pr.Spec.Params))
	for _, currentParam := range pr.Spec.Params {
		rev = append(rev, SpecParameter{
			Name:  currentParam.Name,
			Value: StringPointer(currentParam.Value.StringVal)})
	}
	return rev
}

func (pr *PipelineRun) ToStringForSchedule() string {
	spec, err := json.Marshal(pr.PipelineRun.Spec)
	if err != nil {
		glog.Errorf("Could not marshal the Spec of workflow: %v", pr.PipelineRun)
		return ""
	}
	return string(spec)
}

func (w *PipelineRun) Validate(lint, ignoreEntrypoint bool) error {
	return nil
}

func (pr *PipelineRun) GenerateRetryExecution() (ExecutionSpec, []string, error) {
	if len(pr.Status.Conditions) > 0 {
		switch pr.Status.Conditions[0].Type {
		case "Failed", "Error":
			break
		default:
			return nil, nil, NewBadRequestError(errors.New("workflow cannot be retried"), "Workflow must be Failed/Error to retry")
		}
	}

	// TODO: Fix the below code to retry Tekton task. It may not be possible with the
	//       current implementation because Tekton doesn't have the concept of pipeline
	//       phases.

	newWF := pr.DeepCopy()

	// // Iterate the previous nodes. If it was successful Pod carry it forward
	var podsToDelete []string

	return NewPipelineRun(newWF), podsToDelete, nil
}

func (pr *PipelineRun) CollectionMetrics(retrieveArtifact RetrieveArtifact) ([]*api.RunMetric, []error) {
	runID := pr.ObjectMeta.Labels[LabelKeyWorkflowRunId]
	runMetrics := []*api.RunMetric{}
	partialFailures := []error{}
	for _, taskrunStatus := range pr.Status.TaskRuns {
		nodeMetrics, err := collectTaskRunMetricsOrNil(runID, *taskrunStatus, retrieveArtifact)
		if err != nil {
			partialFailures = append(partialFailures, err)
			continue
		}
		if nodeMetrics != nil {
			if len(runMetrics)+len(nodeMetrics) >= maxMetricsCountLimit {
				leftQuota := maxMetricsCountLimit - len(runMetrics)
				runMetrics = append(runMetrics, nodeMetrics[0:leftQuota]...)
				// TODO(#1426): report the error back to api server to notify user
				log.Errorf("Reported metrics are more than the limit %v", maxMetricsCountLimit)
				break
			}
			runMetrics = append(runMetrics, nodeMetrics...)
		}
	}
	return runMetrics, partialFailures
}

func collectTaskRunMetricsOrNil(
	runID string, taskrunStatus pipelineapi.PipelineRunTaskRunStatus, retrieveArtifact RetrieveArtifact) (
	[]*api.RunMetric, error) {

	defer func() {
		if panicMessage := recover(); panicMessage != nil {
			log.Infof("nodeStatus is not yet created. Panic message: '%v'.", panicMessage)
		}
	}()
	if taskrunStatus.Status == nil ||
		taskrunStatus.Status.TaskRunStatusFields.CompletionTime == nil {
		return nil, nil
	}
	metricsJSON, err := readTaskRunMetricsJSONOrEmpty(runID, taskrunStatus, retrieveArtifact)
	if err != nil || metricsJSON == "" {
		return nil, err
	}

	// Proto json lib requires a proto message before unmarshal data from JSON. We use
	// ReportRunMetricsRequest as a workaround to hold user's metrics, which is a superset of what
	// user can provide.
	reportMetricsRequest := new(api.ReportRunMetricsRequest)
	err = jsonpb.UnmarshalString(metricsJSON, reportMetricsRequest)
	if err != nil {
		// User writes invalid metrics JSON.
		// TODO(#1426): report the error back to api server to notify user
		log.WithFields(log.Fields{
			"run":         runID,
			"node":        taskrunStatus.PipelineTaskName,
			"raw_content": metricsJSON,
			"error":       err.Error(),
		}).Warning("Failed to unmarshal metrics file.")
		return nil, NewCustomError(err, CUSTOM_CODE_PERMANENT,
			"failed to unmarshal metrics file from (%s, %s).", runID, taskrunStatus.PipelineTaskName)
	}
	if reportMetricsRequest.GetMetrics() == nil {
		return nil, nil
	}
	for _, metric := range reportMetricsRequest.GetMetrics() {
		// User metrics just have name and value but no NodeId.
		metric.NodeId = taskrunStatus.PipelineTaskName
	}
	return reportMetricsRequest.GetMetrics(), nil
}

func readTaskRunMetricsJSONOrEmpty(
	runID string, nodeStatus pipelineapi.PipelineRunTaskRunStatus,
	retrieveArtifact RetrieveArtifact) (string, error) {

	artifactRequest := &api.ReadArtifactRequest{
		RunId:        runID,
		NodeId:       nodeStatus.PipelineTaskName,
		ArtifactName: metricsArtifactName,
	}
	artifactResponse, err := retrieveArtifact(artifactRequest)
	if err != nil {
		return "", err
	}
	if artifactResponse == nil || artifactResponse.GetData() == nil || len(artifactResponse.GetData()) == 0 {
		// If artifact is not found or empty content, skip the reporting.
		return "", nil
	}
	archivedFiles, err := ExtractTgz(string(artifactResponse.GetData()))
	if err != nil {
		// Invalid tgz file. This should never happen unless there is a bug in the system and
		// it is a unrecoverable error.
		return "", NewCustomError(err, CUSTOM_CODE_PERMANENT,
			"Unable to extract metrics tgz file read from (%+v): %v", artifactRequest, err)
	}
	//There needs to be exactly one metrics file in the artifact archive. We load that file.
	if len(archivedFiles) == 1 {
		for _, value := range archivedFiles {
			return value, nil
		}
	}
	return "", NewCustomErrorf(CUSTOM_CODE_PERMANENT,
		"There needs to be exactly one metrics file in the artifact archive, but zero or multiple files were found.")
}

func (pr *PipelineRun) NodeStatuses() map[string]NodeStatus {
	// only need taskruns for now, in persistenceagent, it still convert the childreference to taskruns
	// still use status.taskruns to get information for now
	nodeCount := len(pr.Status.TaskRuns)
	rev := make(map[string]NodeStatus, nodeCount)
	for id, node := range pr.Status.TaskRuns {
		// report the node status when the StartTime and CompletionTime are available
		if node.Status.StartTime != nil && node.Status.CompletionTime != nil {
			rev[id] = NodeStatus{
				ID:          id,
				DisplayName: node.PipelineTaskName,
				State:       node.Status.GetCondition(apis.ConditionSucceeded).GetReason(),
				StartTime:   node.Status.StartTime.Unix(),
				CreateTime:  node.Status.StartTime.Unix(),
				FinishTime:  node.Status.CompletionTime.Unix(),
				// no children for a TaskRun task
			}
		}
	}

	return rev
}

func (pr *PipelineRun) HasNodes() bool {
	if len(pr.Status.TaskRuns) == 0 {
		return false
	}
	for _, node := range pr.Status.TaskRuns {
		// report the node status when the StartTime and CompletionTime are available
		if node.Status.StartTime != nil && node.Status.CompletionTime != nil {
			return true
		}
	}
	return false
}

// implementation of ExecutionClientInterface
type PipelineRunClient struct {
	client *prclientset.Clientset
}

func (prc *PipelineRunClient) Execution(namespace string) ExecutionInterface {
	var informer prinformer.PipelineRunInformer
	if namespace == "" {
		informer = prsinformers.NewSharedInformerFactory(prc.client, time.Second*30).
			Tekton().V1().PipelineRuns()
	} else {
		informer = prsinformers.NewFilteredSharedInformerFactory(prc.client, time.Second*30, namespace, nil).
			Tekton().V1().PipelineRuns()
	}

	return &PipelineRunInterface{
		pipelinerunInterface: prc.client.TektonV1().PipelineRuns(namespace),
		informer:             informer,
	}
}

func (prc *PipelineRunClient) Compare(old, new interface{}) bool {
	newWorkflow := new.(*pipelineapi.PipelineRun)
	oldWorkflow := old.(*pipelineapi.PipelineRun)
	// Periodic resync will send update events for all known Workflows.
	// Two different versions of the same WorkflowHistory will always have different RVs.
	return newWorkflow.ResourceVersion != oldWorkflow.ResourceVersion
}

type PipelineRunInterface struct {
	pipelinerunInterface prclientv1.PipelineRunInterface
	informer             prinformer.PipelineRunInformer
}

func (pri *PipelineRunInterface) Create(ctx context.Context, execution ExecutionSpec, opts metav1.CreateOptions) (ExecutionSpec, error) {
	pipelinerun, ok := execution.(*PipelineRun)
	if !ok {
		return nil, fmt.Errorf("execution is not a valid ExecutionSpec for Argo Workflow")
	}

	revPipelineRun, err := pri.pipelinerunInterface.Create(ctx, pipelinerun.PipelineRun, opts)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun,
		Status: TektonStatus{&revPipelineRun.Status, map[string]*pipelineapi.PipelineRunTaskRunStatus{}, map[string]*pipelineapi.PipelineRunRunStatus{}},
	}, nil
}

func (pri *PipelineRunInterface) Update(ctx context.Context, execution ExecutionSpec, opts metav1.UpdateOptions) (ExecutionSpec, error) {
	pipelinerun, ok := execution.(*PipelineRun)
	if !ok {
		return nil, fmt.Errorf("execution is not a valid ExecutionSpec for Argo Workflow")
	}

	revPipelineRun, err := pri.pipelinerunInterface.Update(ctx, pipelinerun.PipelineRun, opts)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun,
		Status: TektonStatus{&revPipelineRun.Status, map[string]*pipelineapi.PipelineRunTaskRunStatus{}, map[string]*pipelineapi.PipelineRunRunStatus{}},
	}, nil
}

func (pri *PipelineRunInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return pri.pipelinerunInterface.Delete(ctx, name, opts)
}

func (pri *PipelineRunInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return pri.pipelinerunInterface.DeleteCollection(ctx, opts, listOpts)
}

func (pri *PipelineRunInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (ExecutionSpec, error) {
	revPipelineRun, err := pri.pipelinerunInterface.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun,
		Status: TektonStatus{&revPipelineRun.Status, map[string]*pipelineapi.PipelineRunTaskRunStatus{}, map[string]*pipelineapi.PipelineRunRunStatus{}},
	}, nil
}

func (pri *PipelineRunInterface) List(ctx context.Context, opts metav1.ListOptions) (*ExecutionSpecList, error) {
	prlist, err := pri.pipelinerunInterface.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	rev := make(ExecutionSpecList, 0, len(prlist.Items))
	for _, pr := range prlist.Items {
		rev = append(rev, &PipelineRun{PipelineRun: &pr,
			Status: TektonStatus{&pr.Status, map[string]*pipelineapi.PipelineRunTaskRunStatus{}, map[string]*pipelineapi.PipelineRunRunStatus{}},
		})
	}
	return &rev, nil
}

func (pri *PipelineRunInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (ExecutionSpec, error) {
	revPipelineRun, err := pri.pipelinerunInterface.Patch(ctx, name, pt, data, opts, subresources...)
	if err != nil {
		return nil, err
	}
	return &PipelineRun{PipelineRun: revPipelineRun,
		Status: TektonStatus{&revPipelineRun.Status, map[string]*pipelineapi.PipelineRunTaskRunStatus{}, map[string]*pipelineapi.PipelineRunRunStatus{}},
	}, nil
}

type PipelineRunInformer struct {
	informer  prinformer.PipelineRunInformer
	clientset *prclientset.Clientset
	factory   prsinformers.SharedInformerFactory
}

func (pri *PipelineRunInformer) AddEventHandler(funcs cache.ResourceEventHandler) {
	pri.informer.Informer().AddEventHandler(funcs)
}

func (pri *PipelineRunInformer) HasSynced() func() bool {
	return pri.informer.Informer().HasSynced
}

func (pri *PipelineRunInformer) Get(namespace string, name string) (ExecutionSpec, bool, error) {
	pipelinerun, err := pri.informer.Lister().PipelineRuns(namespace).Get(name)
	if err != nil {
		return nil, IsNotFound(err), errors.Wrapf(err,
			"Error retrieving PipelineRun (%v) in namespace (%v): %v", name, namespace, err)
	}
	newWorkflow := NewPipelineRun(pipelinerun)

	// Reduce newWorkflow size
	newWorkflow.Spec = pipelineapi.PipelineRunSpec{}
	return newWorkflow, false, nil
}

func (pri *PipelineRunInformer) List(labels *labels.Selector) (ExecutionSpecList, error) {
	pipelineruns, err := pri.informer.Lister().List(*labels)
	if err != nil {
		return nil, err
	}

	rev := make(ExecutionSpecList, 0, len(pipelineruns))
	for _, pipelinerun := range pipelineruns {
		rev = append(rev, NewPipelineRun(pipelinerun))
	}
	return rev, nil
}

func (pri *PipelineRunInformer) InformerFactoryStart(stopCh <-chan struct{}) {
	pri.factory.Start(stopCh)
}
