// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/golang/glog"
	apiV2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// stagesCustomPropertyKey is the single custom_properties key under which all
// reported task stages are nested. Each stage is keyed by its name within the
// value struct.
const stagesCustomPropertyKey = "stages"

// stageNameField is the JSONL field the SDK uses to name a stage. It becomes the
// map key in the stages struct and is not duplicated inside the stage's detail.
const stageNameField = "stage"

// maxStageLineBytes bounds a single JSONL stage report so a stage carrying a
// large message or extra fields is not silently split by the scanner, while
// still rejecting pathologically long lines.
const maxStageLineBytes = 1024 * 1024

// readTaskStages consumes JSONL task-stage reports from r until EOF, which the
// launcher triggers by closing the pipe write-end after the user command exits.
//
// Each line is a JSON object with a required "stage" field; its remaining fields
// (typically message, state, and any user-supplied extras) become the stage's
// detail. Stages accumulate into the shared in-memory
// l.options.Task.StatusMetadata.CustomProperties["stages"] so the normal
// finalize path persists them, and each update is additionally written straight
// through via UpdateTask for live visibility. Malformed lines are discarded.
//
// This goroutine is the sole writer of the task's StatusMetadata while the user
// command runs: the main goroutine is blocked inside CommandExecutor.Run for
// that whole window, and the launcher joins this goroutine before reading or
// flushing StatusMetadata afterward.
func (l *LauncherV2) readTaskStages(ctx context.Context, r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxStageLineBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		fields := map[string]interface{}{}
		if err := json.Unmarshal(line, &fields); err != nil {
			glog.Warningf("task stage reporting: discarding malformed line: %v", err)
			continue
		}
		name, ok := fields[stageNameField].(string)
		if !ok || name == "" {
			glog.Warningf("task stage reporting: discarding line without a %q field", stageNameField)
			continue
		}
		// The stage name is the map key, so it is not repeated inside the detail.
		delete(fields, stageNameField)
		detail, err := structpb.NewStruct(fields)
		if err != nil {
			glog.Warningf("task stage reporting: discarding stage %q with non-serializable fields: %v", name, err)
			continue
		}
		l.upsertStage(name, structpb.NewStructValue(detail))
		if err := l.persistStages(ctx); err != nil {
			glog.Warningf("task stage reporting: failed to persist stage %q: %v", name, err)
		}
	}
	if err := scanner.Err(); err != nil {
		glog.Warningf("task stage reporting: error reading stage pipe: %v", err)
	}
}

// upsertStage records or replaces a single stage in the in-memory task's
// StatusMetadata, keyed by stage name under the "stages" custom property. Any
// pre-existing custom properties (for example those set by the driver) are left
// untouched.
func (l *LauncherV2) upsertStage(name string, detail *structpb.Value) {
	if l.options.Task.StatusMetadata == nil {
		l.options.Task.StatusMetadata = &apiV2.PipelineTask_StatusMetadata{}
	}
	if l.options.Task.StatusMetadata.CustomProperties == nil {
		l.options.Task.StatusMetadata.CustomProperties = map[string]*structpb.Value{}
	}
	stages := l.options.Task.StatusMetadata.CustomProperties[stagesCustomPropertyKey].GetStructValue()
	if stages == nil {
		stages = &structpb.Struct{Fields: map[string]*structpb.Value{}}
		l.options.Task.StatusMetadata.CustomProperties[stagesCustomPropertyKey] = structpb.NewStructValue(stages)
	}
	if stages.Fields == nil {
		stages.Fields = map[string]*structpb.Value{}
	}
	stages.Fields[name] = detail
}

// persistStages writes the current accumulated stages to the backend with a
// minimal task update. Sending only TaskId, RunId, and StatusMetadata lets the
// task store's selective, row-locked update touch just the StatusMetadata
// column, leaving state, parameters, pods, and other fields untouched.
func (l *LauncherV2) persistStages(ctx context.Context) error {
	taskID := l.options.Task.GetTaskId()
	if taskID == "" {
		return nil
	}
	runID := l.options.Task.GetRunId()
	statusMetadata := proto.Clone(l.options.Task.StatusMetadata).(*apiV2.PipelineTask_StatusMetadata)
	_, err := l.clientManager.KFPAPIClient().UpdateTask(ctx, &apiV2.UpdateTaskRequest{
		TaskId: taskID,
		RunId:  runID,
		Task: &apiV2.PipelineTask{
			TaskId:         taskID,
			RunId:          runID,
			StatusMetadata: statusMetadata,
		},
	})
	return err
}
