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

package mlmd

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
	"sigs.k8s.io/yaml"
)

const cacheBackfillWindowSeconds int64 = 90 * 24 * 60 * 60

type cacheRun struct {
	name       string
	spec       string
	parsed     *pipelinespec.PipelineSpec
	deployment *pipelinespec.PipelineDeploymentConfig
	parseErr   error
	prepared   bool
}

func loadCacheRuns(ctx context.Context, db *gorm.DB) (map[string]*cacheRun, error) {
	rows, err := db.WithContext(ctx).Raw("SELECT UUID, PipelineName, PipelineSpecManifest FROM run_details").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := map[string]*cacheRun{}
	for rows.Next() {
		var id string
		var name, spec sql.NullString
		if err := rows.Scan(&id, &name, &spec); err != nil {
			return nil, err
		}
		runs[id] = &cacheRun{name: name.String, spec: spec.String}
	}
	return runs, rows.Err()
}

// cacheCutoff is created once under the migration advisory lock. Keeping the
// marker makes an interrupted run and a later CLI true-up use the same window.
func cacheCutoff(ctx context.Context, db *gorm.DB, migrating bool) (int64, error) {
	if migrating {
		if err := db.WithContext(ctx).Exec("CREATE TABLE IF NOT EXISTS mlmd_migration_state (ID tinyint NOT NULL PRIMARY KEY, CacheCutoffInSec bigint NOT NULL) ENGINE=InnoDB").Error; err != nil {
			return 0, err
		}
		if err := db.WithContext(ctx).Exec("INSERT IGNORE INTO mlmd_migration_state (ID, CacheCutoffInSec) VALUES (1, UNIX_TIMESTAMP())").Error; err != nil {
			return 0, err
		}
	}
	exists, err := tableExists(ctx, db, "mlmd_migration_state")
	if err != nil || !exists {
		return 0, err
	}
	var cutoff int64
	if err := db.WithContext(ctx).Raw("SELECT CacheCutoffInSec FROM mlmd_migration_state WHERE ID=1").Row().Scan(&cutoff); err != nil {
		return 0, err
	}
	return cutoff, nil
}

func loadNativeCacheTimestamps(ctx context.Context, db *gorm.DB, snapshot *Snapshot, legacy map[int64]LegacyTask) error {
	rows, err := db.WithContext(ctx).Raw("SELECT UUID, CreatedAtInSec FROM tasks").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	created := map[string]int64{}
	for rows.Next() {
		var id string
		var timestamp int64
		if err := rows.Scan(&id, &timestamp); err != nil {
			return err
		}
		created[id] = timestamp
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, execution := range snapshot.Executions {
		if timestamp := created[ExecutionUUID(execution.GetId())]; timestamp > 0 {
			legacy[execution.GetId()] = LegacyTask{CreatedAtInSec: timestamp}
		}
	}
	return nil
}

func decodeLegacyParameters(value *pb.Value) (map[string]*structpb.Value, error) {
	if value == nil {
		return nil, fmt.Errorf("resolved execution inputs are absent")
	}
	if value.GetStructValue() != nil {
		return value.GetStructValue().GetFields(), nil
	}
	encoded, found := strings.CutPrefix(value.GetStringValue(), "mlmd-struct::")
	if !found {
		return nil, fmt.Errorf("resolved execution inputs have an unknown encoding")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode resolved execution inputs: %w", err)
	}
	decoded := &structpb.Struct{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		return nil, fmt.Errorf("parse resolved execution inputs: %w", err)
	}
	return decoded.GetFields(), nil
}

func cacheTaskSpec(spec *pipelinespec.PipelineSpec, name string) (*pipelinespec.PipelineTaskSpec, *pipelinespec.ComponentSpec, error) {
	var selected *pipelinespec.PipelineTaskSpec
	var component *pipelinespec.ComponentSpec
	visiting := map[*pipelinespec.ComponentSpec]bool{}
	var visit func(*pipelinespec.ComponentSpec) error
	visit = func(parent *pipelinespec.ComponentSpec) error {
		if visiting[parent] {
			return fmt.Errorf("saved pipeline spec has a component cycle")
		}
		visiting[parent] = true
		defer delete(visiting, parent)
		for _, task := range parent.GetDag().GetTasks() {
			child := spec.GetComponents()[task.GetComponentRef().GetName()]
			if child == nil {
				return fmt.Errorf("pipeline component %q is absent", task.GetComponentRef().GetName())
			}
			if task.GetTaskInfo().GetName() == name && child.GetExecutorLabel() != "" {
				if selected != nil && !proto.Equal(selected, task) {
					return fmt.Errorf("task name %q is ambiguous in the saved pipeline spec", name)
				}
				selected, component = task, child
			}
			if child.GetDag() != nil {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(spec.GetRoot()); err != nil {
		return nil, nil, err
	}
	if selected == nil {
		return nil, nil, fmt.Errorf("task %q is absent from the saved pipeline spec", name)
	}
	return selected, component, nil
}

func parseCacheSpec(source string) (*pipelinespec.PipelineSpec, *pipelinespec.PipelineDeploymentConfig, error) {
	if source == "" {
		return nil, nil, fmt.Errorf("saved pipeline spec is absent")
	}
	data, err := yaml.YAMLToJSON([]byte(source))
	if err != nil {
		return nil, nil, fmt.Errorf("decode saved pipeline spec: %w", err)
	}
	spec := &pipelinespec.PipelineSpec{}
	if err := protojson.Unmarshal(data, spec); err != nil {
		return nil, nil, fmt.Errorf("parse saved pipeline spec: %w", err)
	}
	deployment := &pipelinespec.PipelineDeploymentConfig{}
	encoded, err := protojson.Marshal(spec.GetDeploymentSpec())
	if err != nil {
		return nil, nil, err
	}
	if err := protojson.Unmarshal(encoded, deployment); err != nil {
		return nil, nil, fmt.Errorf("parse deployment spec: %w", err)
	}
	return spec, deployment, nil
}

func reconstructNativeFingerprint(execution *pb.Execution, run *cacheRun) (string, error) {
	if run == nil {
		return "", fmt.Errorf("saved run is absent")
	}
	if !run.prepared {
		run.parsed, run.deployment, run.parseErr = parseCacheSpec(run.spec)
		run.prepared = true
	}
	if run.parseErr != nil {
		return "", run.parseErr
	}
	spec, deployment := run.parsed, run.deployment
	if run.name == "" {
		run.name = spec.GetPipelineInfo().GetName()
	}
	if run.name == "" {
		return "", fmt.Errorf("pipeline name is absent")
	}
	name := propertyString(execution.GetCustomProperties(), "task_name")
	task, component, err := cacheTaskSpec(spec, name)
	if err != nil {
		return "", err
	}
	if !task.GetCachingOptions().GetEnableCache() {
		return "", fmt.Errorf("saved task does not enable caching")
	}
	legacy := propertyString(execution.GetCustomProperties(), "cache_fingerprint")
	if task.GetCachingOptions().GetCacheKey() != "" {
		return "", fmt.Errorf("custom cache key used a pipeline-scoped legacy lookup but native lookup is namespace-scoped")
	}
	container := deployment.GetExecutors()[component.GetExecutorLabel()].GetContainer()
	if container == nil {
		return "", fmt.Errorf("saved container spec is absent")
	}
	if len(component.GetInputDefinitions().GetArtifacts()) != 0 {
		return "", fmt.Errorf("resolved input artifact identities are unavailable for key reconstruction")
	}
	parameters, err := decodeLegacyParameters(execution.GetCustomProperties()["inputs"])
	if err != nil {
		return "", err
	}
	inputs := &pipelinespec.ExecutorInput_Inputs{ParameterValues: parameters}
	outputs := &pipelinespec.ExecutorInput_Outputs{
		Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{},
		Artifacts:  map[string]*pipelinespec.ArtifactList{},
	}
	parameterTypes := map[string]string{}
	for key, parameter := range component.GetOutputDefinitions().GetParameters() {
		outputs.Parameters[key] = &pipelinespec.ExecutorInput_OutputParameter{}
		parameterTypes[key] = parameter.GetParameterType().String()
	}
	for key, artifact := range component.GetOutputDefinitions().GetArtifacts() {
		outputs.Artifacts[key] = &pipelinespec.ArtifactList{Artifacts: []*pipelinespec.RuntimeArtifact{{
			Name: key, Type: artifact.GetArtifactType(), Metadata: artifact.GetMetadata(),
		}}}
	}
	commands := append(append([]string{}, container.GetCommand()...), container.GetArgs()...)
	makeKey := func(args []string) (string, error) {
		key, err := cacheutils.GenerateCacheKey(inputs, outputs, parameterTypes, args, container.GetImage(), nil, container.GetEnv())
		if err != nil {
			return "", err
		}
		return cacheutils.GenerateFingerPrint(key)
	}
	oldKey, err := makeKey(commands)
	if err != nil {
		return "", err
	}
	if oldKey != legacy {
		outputs.Artifacts["executor-logs"] = &pipelinespec.ArtifactList{Artifacts: []*pipelinespec.RuntimeArtifact{{
			Name: "executor-logs", Type: &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Artifact"}},
		}}}
		withLogs, err := makeKey(commands)
		if err != nil {
			return "", err
		}
		if withLogs == legacy {
			oldKey = withLogs
		} else {
			delete(outputs.Artifacts, "executor-logs")
		}
	}
	if oldKey != legacy {
		return "", fmt.Errorf("reconstructed inputs do not reproduce source fingerprint")
	}
	return makeKey(append(commands, "__kfp_pipeline_name="+run.name))
}

func preserveLegacyFingerprint(task *model.Task, fingerprint string) {
	if fingerprint == "" {
		return
	}
	if task.StatusMetadata == nil {
		task.StatusMetadata = model.JSONData{}
	}
	custom, _ := task.StatusMetadata["customProperties"].(map[string]any)
	if custom == nil {
		custom = map[string]any{}
	}
	custom["legacy_cache_fingerprint"] = fingerprint
	task.StatusMetadata["customProperties"] = custom
	task.Fingerprint = ""
}

func backfillNativeCacheKeys(plan *Plan, snapshot *Snapshot, legacy map[int64]LegacyTask, runs map[string]*cacheRun, cutoff int64) int {
	byTask := map[string]*model.Task{}
	backfilled := 0
	for index := range plan.Tasks {
		byTask[plan.Tasks[index].UUID] = &plan.Tasks[index]
	}
	for _, execution := range snapshot.Executions {
		if execution.GetLastKnownState() != pb.Execution_COMPLETE && execution.GetLastKnownState() != pb.Execution_CACHED {
			continue
		}
		fingerprint := propertyString(execution.GetCustomProperties(), "cache_fingerprint")
		if fingerprint == "" {
			continue
		}
		task := byTask[ExecutionUUID(execution.GetId())]
		if task == nil {
			continue
		}
		preserveLegacyFingerprint(task, fingerprint)
		if execution.GetLastKnownState() != pb.Execution_COMPLETE || task.State != model.TaskStatus(apiv2.PipelineTask_SUCCEEDED) {
			continue
		}
		created := legacy[execution.GetId()].CreatedAtInSec
		if created <= 0 {
			cacheSkip(plan, execution.GetId(), "original cache entry creation time is absent")
			continue
		}
		if cutoff == 0 || created > cutoff || created < cutoff-cacheBackfillWindowSeconds {
			continue
		}
		key, err := reconstructNativeFingerprint(execution, runs[task.RunUUID])
		if err != nil {
			cacheSkip(plan, execution.GetId(), err.Error())
			continue
		}
		task.Fingerprint = key
		backfilled++
	}
	return backfilled
}

func cacheSkip(plan *Plan, id int64, reason string) {
	log.Printf("ERROR: skipped native cache key for MLMD execution %d: %s", id, reason)
	plan.Skipped = append(plan.Skipped, Skip{Kind: "cache_key", ID: id, Reason: reason})
}
