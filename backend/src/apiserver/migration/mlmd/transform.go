// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package mlmd contains the shared transformations for migrating an MLMD store.
package mlmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

// ExecutionUUID is the stable native identity of an MLMD execution.
func ExecutionUUID(id int64) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte("execution:"+strconv.FormatInt(id, 10))).String()
}

// ArtifactUUID is the stable native identity of an MLMD artifact.
func ArtifactUUID(id int64) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:"+strconv.FormatInt(id, 10))).String()
}

// MetricUUID is the stable identity of a non-primary metric from an MLMD artifact.
func MetricUUID(id int64, name string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte("artifact:"+strconv.FormatInt(id, 10)+":metric:"+name)).String()
}

func propertyString(values map[string]*pb.Value, key string) string {
	if value := values[key]; value != nil {
		return value.GetStringValue()
	}
	return ""
}

func parentID(execution *pb.Execution) int64 {
	if value := execution.GetCustomProperties()["parent_dag_id"]; value != nil {
		return value.GetIntValue()
	}
	return 0
}

func taskType(execution *pb.Execution) (model.TaskType, error) {
	switch execution.GetType() {
	case "system.ContainerExecution":
		return model.TaskType(apiv2.PipelineTask_RUNTIME), nil
	case "system.ImporterExecution":
		return model.TaskType(apiv2.PipelineTask_IMPORTER), nil
	case "system.DAGExecution":
		name := propertyString(execution.GetCustomProperties(), "task_name")
		if name == "" {
			name = execution.GetName()
		}
		exitProperty := execution.GetCustomProperties()["is_exit_handler"]
		if exitProperty == nil {
			exitProperty = execution.GetCustomProperties()["exit_handler"]
		}
		isExitHandler := exitProperty != nil && (exitProperty.GetBoolValue() || exitProperty.GetStringValue() == "true")
		switch {
		case isExitHandler, strings.HasPrefix(name, "exit-handler-"), strings.HasPrefix(name, "on-exit-"), strings.HasPrefix(name, "onexit-"):
			return model.TaskType(apiv2.PipelineTask_EXIT_HANDLER), nil
		case strings.HasPrefix(name, "condition-branches-"):
			return model.TaskType(apiv2.PipelineTask_CONDITION_BRANCH), nil
		case strings.HasPrefix(name, "condition-"):
			return model.TaskType(apiv2.PipelineTask_CONDITION), nil
		case execution.GetCustomProperties()["iteration_count"] != nil:
			return model.TaskType(apiv2.PipelineTask_LOOP), nil
		case parentID(execution) == 0:
			return model.TaskType(apiv2.PipelineTask_ROOT), nil
		default:
			return model.TaskType(apiv2.PipelineTask_DAG), nil
		}
	default:
		return 0, fmt.Errorf("execution %d has unsupported type %q", execution.GetId(), execution.GetType())
	}
}

func taskState(state pb.Execution_State) model.TaskStatus {
	switch state {
	case pb.Execution_RUNNING:
		return model.TaskStatus(apiv2.PipelineTask_RUNNING)
	case pb.Execution_COMPLETE:
		return model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)
	case pb.Execution_CACHED:
		return model.TaskStatus(apiv2.PipelineTask_CACHED)
	case pb.Execution_FAILED, pb.Execution_CANCELED:
		return model.TaskStatus(apiv2.PipelineTask_FAILED)
	default:
		return model.TaskStatus(apiv2.PipelineTask_RUNTIME_STATE_UNSPECIFIED)
	}
}

func parameterRows(value *pb.Value, kind string) model.JSONSlice {
	if value == nil || value.GetStructValue() == nil {
		return nil
	}
	properties := value.GetStructValue().AsMap()
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make(model.JSONSlice, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, map[string]any{"parameterKey": key, "type": kind, "value": properties[key]})
	}
	return rows
}

// TransformExecution converts one MLMD execution. ParentTaskUUID is deliberately
// unset until all task rows exist, so the self-referencing foreign key is valid.
func TransformExecution(execution *pb.Execution, runUUID, namespace string) (model.Task, int64, error) {
	typeValue, err := taskType(execution)
	if err != nil {
		return model.Task{}, 0, err
	}
	properties := execution.GetCustomProperties()
	name := propertyString(properties, "task_name")
	if name == "" {
		name = execution.GetName()
	}
	displayName := propertyString(properties, "display_name")
	if len(name) > 128 || len(displayName) > 128 || len(namespace) > 63 {
		return model.Task{}, 0, fmt.Errorf("execution %d exceeds native task name, display name, or namespace width", execution.GetId())
	}
	attributes := model.JSONData{}
	for source, target := range map[string]string{"iteration_count": "iterationCount", "iteration_index": "iterationIndex"} {
		if value := properties[source]; value != nil {
			attributes[target] = strconv.FormatInt(value.GetIntValue(), 10)
		}
	}
	fingerprint := ""
	if execution.GetLastKnownState() == pb.Execution_COMPLETE || execution.GetLastKnownState() == pb.Execution_CACHED {
		fingerprint = propertyString(properties, "cache_fingerprint")
	}
	pods := model.JSONSlice{}
	if podName := propertyString(properties, "pod_name"); podName != "" {
		pod := map[string]any{"name": podName, "type": "EXECUTOR"}
		if uid := propertyString(properties, "pod_uid"); uid != "" {
			pod["uid"] = uid
		}
		pods = append(pods, pod)
	}
	var status model.JSONData
	if execution.GetLastKnownState() == pb.Execution_CACHED {
		if sourceID, err := strconv.ParseInt(propertyString(properties, "cached_execution_id"), 10, 64); err == nil && sourceID > 0 {
			status = model.JSONData{"customProperties": map[string]any{"cached_task_id": ExecutionUUID(sourceID)}}
		}
	}
	task := model.Task{
		UUID: ExecutionUUID(execution.GetId()), RunUUID: runUUID, Namespace: namespace,
		Name: name, DisplayName: displayName,
		Type: typeValue, TypeAttrs: attributes, State: taskState(execution.GetLastKnownState()),
		Pods: pods, StatusMetadata: status,
		InputParameters:  parameterRows(properties["inputs"], "RUNTIME_VALUE_INPUT"),
		OutputParameters: parameterRows(properties["outputs"], "OUTPUT"),
		CreatedAtInSec:   execution.GetCreateTimeSinceEpoch() / 1000,
	}
	preserveLegacyFingerprint(&task, fingerprint)
	return task, parentID(execution), nil
}

// TransformArtifact expands a metrics artifact into deterministic native rows.
// The first metric retains the source artifact UUID for event lineage.
func TransformArtifact(artifact *pb.Artifact, namespace string) []model.Artifact {
	uri := artifact.GetUri()
	uriHash := ""
	if uri != "" {
		sum := sha256.Sum256([]byte(uri))
		uriHash = hex.EncodeToString(sum[:])
	}
	base := model.Artifact{
		UUID: ArtifactUUID(artifact.GetId()), Namespace: namespace,
		URI: &uri, URIHash: uriHash,
		Name:            propertyString(artifact.GetCustomProperties(), "display_name"),
		CreatedAtInSec:  artifact.GetCreateTimeSinceEpoch() / 1000,
		LastUpdateInSec: artifact.GetLastUpdateTimeSinceEpoch() / 1000,
		Metadata:        model.JSONData{},
	}
	properties := artifact.GetCustomProperties()
	for name, value := range properties {
		if value == nil {
			continue
		}
		switch typed := value.GetValue().(type) {
		case *pb.Value_StringValue:
			base.Metadata[name] = typed.StringValue
		case *pb.Value_IntValue:
			base.Metadata[name] = typed.IntValue
		case *pb.Value_DoubleValue:
			base.Metadata[name] = typed.DoubleValue
		case *pb.Value_BoolValue:
			base.Metadata[name] = typed.BoolValue
		case *pb.Value_StructValue:
			base.Metadata[name] = typed.StructValue.AsMap()
		case *pb.Value_ProtoValue:
			base.Metadata[name] = map[string]any{"typeUrl": typed.ProtoValue.GetTypeUrl(), "valueBase64": base64.StdEncoding.EncodeToString(typed.ProtoValue.GetValue())}
		}
	}
	switch artifact.GetType() {
	case "system.Model":
		base.Type = model.ArtifactType(apiv2.Artifact_Model)
	case "system.Dataset":
		base.Type = model.ArtifactType(apiv2.Artifact_Dataset)
	case "system.Metrics":
		base.Type = model.ArtifactType(apiv2.Artifact_Metric)
	case "system.HTML":
		base.Type = model.ArtifactType(apiv2.Artifact_HTML)
	case "system.Markdown":
		base.Type = model.ArtifactType(apiv2.Artifact_Markdown)
	case "system.ClassificationMetrics":
		base.Type = model.ArtifactType(apiv2.Artifact_ClassificationMetric)
	case "system.SlicedClassificationMetrics":
		base.Type = model.ArtifactType(apiv2.Artifact_SlicedClassificationMetric)
	default:
		base.Type = model.ArtifactType(apiv2.Artifact_Artifact)
	}
	if artifact.GetType() != "system.Metrics" {
		return []model.Artifact{base}
	}
	metricNames := []string{}
	for name, value := range properties {
		if value == nil {
			continue
		}
		switch value.GetValue().(type) {
		case *pb.Value_DoubleValue, *pb.Value_IntValue:
			metricNames = append(metricNames, name)
		}
	}
	sort.Strings(metricNames)
	if len(metricNames) == 0 {
		return []model.Artifact{base}
	}
	rows := make([]model.Artifact, 0, len(metricNames))
	for index, name := range metricNames {
		row := base
		row.Name = name
		value := properties[name]
		number := value.GetDoubleValue()
		if _, ok := value.GetValue().(*pb.Value_IntValue); ok {
			number = float64(value.GetIntValue())
		}
		row.NumberValue = &number
		if index != 0 {
			row.UUID = MetricUUID(artifact.GetId(), name)
		}
		rows = append(rows, row)
	}
	return rows
}

// TransformEvent links an MLMD event to the primary native artifact row.
func TransformEvent(event *pb.Event, runUUID, artifactKey string, producer *apiv2.IOProducer) (model.ArtifactTask, error) {
	taskUUID := ExecutionUUID(event.GetExecutionId())
	artifactUUID := ArtifactUUID(event.GetArtifactId())
	iteration := model.ArtifactTaskNoIteration
	var producerData model.JSONData
	if producer != nil {
		var err error
		producerData, err = model.ProtoMessageToJSONData(producer)
		if err != nil {
			return model.ArtifactTask{}, err
		}
		if producer.Iteration != nil {
			iteration = producer.GetIteration()
		}
	}
	var ioType apiv2.IOType
	switch event.GetType() {
	case pb.Event_INPUT:
		ioType = apiv2.IOType_TASK_OUTPUT_INPUT
	case pb.Event_OUTPUT:
		ioType = apiv2.IOType_OUTPUT
	default:
		return model.ArtifactTask{}, fmt.Errorf("event for execution %d has unsupported type %s", event.GetExecutionId(), event.GetType())
	}
	identity, err := json.Marshal([]any{"artifact-task", artifactUUID, taskUUID, int32(ioType), iteration, artifactKey})
	if err != nil {
		return model.ArtifactTask{}, err
	}
	return model.ArtifactTask{
		UUID:       uuid.NewSHA1(uuid.NameSpaceDNS, identity).String(),
		ArtifactID: artifactUUID, TaskID: taskUUID, RunUUID: runUUID,
		Type: model.IOType(ioType), Iteration: iteration,
		Producer: producerData, ArtifactKey: artifactKey,
	}, nil
}
