// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package history

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/gorm"
)

// DecodeNamespaceArchive validates either the native wire format or the frozen
// release-2.18 format, converting the latter entirely in memory. The source SQL
// schema fingerprint is not a destination schema: only the explicit, validated
// legacy conversion can bind an archive to the destination's current schema.
func DecodeNamespaceArchive(db *gorm.DB, archive []byte, namespace, runtimeNamespace string, externalCatalog bool) (*NamespaceBundle, []string, error) {
	if len(archive) > transfer.MaxArchiveBytes {
		return nil, nil, util.NewInvalidInputError("archive exceeds 256 MiB")
	}
	var header struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(archive, &header); err != nil {
		return nil, nil, util.NewInvalidInputError("invalid archive JSON")
	}
	if header.Format != TransferFormat && header.Format != legacyArchiveFormat {
		return nil, nil, util.NewInvalidInputError("unsupported namespace archive format")
	}
	if header.Format == TransferFormat {
		var bundle NamespaceBundle
		if err := decodeArchiveJSON(archive, &bundle); err != nil {
			return nil, nil, err
		}
		if err := restoreNativeRuntimeParameters(&bundle); err != nil {
			return nil, nil, err
		}
		if err := ValidateNamespace(&bundle, namespace, runtimeNamespace); err != nil {
			return nil, nil, err
		}
		schema, err := schemaSignature(db)
		if err != nil {
			return nil, nil, err
		}
		if bundle.Schema != schema {
			return nil, nil, util.NewInvalidInputError("archive schema does not match this server and destination")
		}
		return &bundle, nil, nil
	}
	var old legacyArchive
	if err := decodeArchiveJSON(archive, &old); err != nil {
		return nil, nil, err
	}
	checksum := old.Digest
	old.Digest = ""
	computed, err := digest(old)
	if err != nil || checksum == "" || checksum != computed {
		return nil, nil, util.NewInvalidInputError("release-2.18 archive digest does not match its contents")
	}
	if err = validateLegacyArchive(&old, namespace, runtimeNamespace); err != nil {
		return nil, nil, err
	}
	if !externalCatalog && len(old.CatalogDefaults) > 0 {
		return nil, nil, util.NewInvalidInputError("release-2.18 archive has pinned Kubernetes catalog defaults; import into a Kubernetes catalog")
	}
	bundle, err := convertLegacyArchive(&old)
	if err != nil {
		return nil, nil, util.NewInvalidInputError("cannot convert legacy archive: %v", err)
	}
	// Empty logical namespaces in old single-user rows are bound to the
	// configured runtime namespace only after their archive ownership is proven.
	if old.Namespace == "" {
		for i := range old.Runs {
			old.Runs[i].Run.Namespace = old.RuntimeNamespace
			for j := range old.Runs[i].Tasks {
				old.Runs[i].Tasks[j].Namespace = old.RuntimeNamespace
			}
		}
	}
	bundle.Entries, err = convertLegacyMLMD(old.Source, old.RuntimeNamespace, old.Runs, old.Metadata, bundle.Entries)
	if err != nil {
		return nil, nil, util.NewInvalidInputError("cannot convert legacy metadata: %v", err)
	}
	if err = ValidateNamespace(bundle, namespace, runtimeNamespace); err != nil {
		return nil, nil, err
	}
	bundle.Schema, err = schemaSignature(db)
	if err != nil {
		return nil, nil, err
	}
	return bundle, nil, nil
}

func decodeArchiveJSON(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return util.NewInvalidInputError("invalid archive: %v", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return util.NewInvalidInputError("archive must contain exactly one JSON value")
	}
	return nil
}

func legacyID(id string) bool {
	return id != "" && len(id) <= 191 && utf8.ValidString(id) && !strings.ContainsRune(id, 0)
}

func validateLegacyArchive(b *legacyArchive, namespace, runtimeNamespace string) error {
	invalid := util.NewInvalidInputError
	if b.Format != legacyArchiveFormat {
		return invalid("unsupported legacy archive version")
	}
	if b.RuntimeParameters == nil || len(b.RuntimeParameters.Runs) != len(b.Runs) || len(b.RuntimeParameters.Schedules) != len(b.Schedules) {
		return invalid("legacy v2 runtime parameters must cover every run and schedule")
	}
	for _, r := range b.Runs {
		if value, ok := b.RuntimeParameters.Runs[r.Run.UUID]; !ok || !validRuntimeParameters(value) {
			return invalid("legacy v2 run runtime parameters are missing")
		}
	}
	for _, j := range b.Schedules {
		if value, ok := b.RuntimeParameters.Schedules[j.UUID]; !ok || !validRuntimeParameters(value) {
			return invalid("legacy v2 schedule runtime parameters are missing")
		}
	}
	if _, err := uuid.Parse(b.Source); err != nil {
		return invalid("invalid legacy source installation identity")
	}
	if decoded, err := hex.DecodeString(b.Schema); err != nil || len(decoded) != 32 {
		return invalid("invalid legacy schema fingerprint")
	}
	if b.Namespace != namespace || b.RuntimeNamespace != runtimeNamespace || (namespace != "" && runtimeNamespace != namespace) || len(namespace) > 63 || len(runtimeNamespace) > 63 {
		return invalid("destination namespace must match archive namespace and runtime namespace")
	}
	if len(b.Runs) > MaxTransferRuns || len(b.Experiments)+len(b.Pipelines)+len(b.Versions)+len(b.Schedules) > MaxTransferObjects {
		return invalid("archive exceeds native transfer limits; export a smaller completion interval")
	}
	rows := len(b.References)
	for _, h := range b.Runs {
		rows += len(h.Tasks) + len(h.Metrics)
	}
	for _, nodes := range b.Metadata {
		rows += len(nodes)
	}
	if rows > MaxTransferRows {
		return invalid("legacy archive exceeds native row limits; export a smaller completion interval")
	}
	sets := map[string]map[string]bool{}
	add := func(kind, id string) error {
		if !legacyID(id) {
			return invalid("invalid legacy %s ID", kind)
		}
		if sets[kind] == nil {
			sets[kind] = map[string]bool{}
		}
		if sets[kind][id] {
			return invalid("duplicate legacy %s ID", kind)
		}
		sets[kind][id] = true
		return nil
	}
	for _, x := range b.Experiments {
		if x.Namespace != namespace {
			return invalid("legacy experiment has invalid ownership or ID")
		}
		if err := add("experiment", x.UUID); err != nil {
			return err
		}
	}
	for _, x := range b.Pipelines {
		if x.Namespace != namespace || x.Status != "READY" || len(x.UUID) > 64 {
			return invalid("legacy pipeline must be ready in the archive namespace")
		}
		if err := add("pipeline", x.UUID); err != nil {
			return err
		}
	}
	versions := map[string]legacyPipelineVersion{}
	for _, x := range b.Versions {
		if !sets["pipeline"][x.PipelineId] || x.Status != "READY" || x.PipelineSpec == "" || x.PipelineSpecURI != "" || !reflect.DeepEqual(x.Pipeline, legacyPipeline{}) {
			return invalid("legacy pipeline version requires its parent and an inline definition without a source URI or nested pipeline")
		}
		if err := add("version", x.UUID); err != nil {
			return err
		}
		versions[x.UUID] = x
	}
	checkSpec := func(s legacyPipelineSpec) error {
		if s.PipelineId != "" && !sets["pipeline"][s.PipelineId] {
			return invalid("legacy pipeline reference is outside archive")
		}
		if s.PipelineVersionId != "" {
			v, ok := versions[s.PipelineVersionId]
			if !ok || (s.PipelineId != "" && v.PipelineId != s.PipelineId) {
				return invalid("legacy version reference is outside its archive pipeline")
			}
		}
		return nil
	}
	for _, j := range b.Schedules {
		if !sets["experiment"][j.ExperimentId] || (j.Namespace != namespace && j.Namespace != runtimeNamespace) || len(j.ResourceReferences) > 0 {
			return invalid("legacy schedule has invalid ownership or hydrated references")
		}
		if err := add("schedule", j.UUID); err != nil {
			return err
		}
		if err := checkSpec(j.legacyPipelineSpec); err != nil {
			return err
		}
	}
	for _, h := range b.Runs {
		r := h.Run
		if !sets["experiment"][r.ExperimentId] || (r.Namespace != namespace && r.Namespace != runtimeNamespace) || r.ImportedFrom != nil || r.ImportDigest != nil || len(r.Metrics) > 0 || len(r.ResourceReferences) > 0 || len(r.TaskDetails) > 0 {
			return invalid("legacy run has invalid ownership, import markers, or hydrated records")
		}
		if r.RecurringRunId != "" && !sets["schedule"][r.RecurringRunId] {
			return invalid("legacy run references a missing schedule")
		}
		if err := add("run", r.UUID); err != nil {
			return err
		}
		if err := checkSpec(r.legacyPipelineSpec); err != nil {
			return err
		}
		state := model.RuntimeState(r.State)
		if state == "" {
			state = model.RuntimeState(r.Conditions)
		}
		switch state.ToV2() {
		case model.RuntimeStateSucceeded, model.RuntimeStateFailed, model.RuntimeStateCanceled, model.RuntimeStateSkipped:
		default:
			return invalid("legacy run is not terminal")
		}
		if r.FinishedAtInSec <= 0 {
			return invalid("legacy run is not completed")
		}
		if r.StateHistoryString != "" {
			if err := validateLegacyRunStateHistory([]byte(r.StateHistoryString)); err != nil {
				return err
			}
		}
		if len(r.StateHistory) > 0 {
			data, err := json.Marshal(r.StateHistory)
			if err != nil {
				return invalid("invalid legacy run state history")
			}
			if err = validateLegacyRunStateHistory(data); err != nil {
				return err
			}
		}

		tasks := map[string]legacyTask{}
		for _, task := range h.Tasks {
			if task.RunID != r.UUID || task.Namespace != r.Namespace || !reflect.DeepEqual(task.Run, legacyRun{}) {
				return invalid("legacy task belongs to a different run or namespace")
			}
			if err := add("task", task.UUID); err != nil {
				return err
			}
			tasks[task.UUID] = task
		}
		if err := validateLegacyTaskParents(tasks); err != nil {
			return err
		}
		for _, task := range h.Tasks {
			if task.Payload != "" {
				var payload legacyTask
				if err := decodeArchiveJSON([]byte(task.Payload), &payload); err != nil {
					return err
				}
				if payload.UUID != task.UUID || payload.RunID != task.RunID || payload.Namespace != task.Namespace {
					return invalid("legacy task payload identity differs from its row")
				}
			}
		}
		metrics := map[string]bool{}
		for _, m := range h.Metrics {
			key := m.NodeID + "\x00" + m.Name
			if m.RunUUID != r.UUID || !legacyID(m.NodeID) || !legacyID(m.Name) || metrics[key] {
				return invalid("legacy metric has invalid ownership or duplicate identity")
			}
			metrics[key] = true
		}
	}
	kinds := map[string]string{"Experiment": "experiment", "pipeline": "pipeline", "PipelineVersion": "version", "Job": "schedule", "RecurringRun": "schedule", "Run": "run"}
	expected := map[string]map[string]string{}
	own := func(kind, id, experiment, pipeline, version, schedule string) {
		expected[kind+"\x00"+id] = map[string]string{"Experiment": experiment, "pipeline": pipeline, "PipelineVersion": version, "Job": schedule, "RecurringRun": schedule}
	}
	for _, r := range b.Runs {
		own("Run", r.Run.UUID, r.Run.ExperimentId, r.Run.PipelineId, r.Run.PipelineVersionId, r.Run.RecurringRunId)
	}
	for _, j := range b.Schedules {
		own("Job", j.UUID, j.ExperimentId, j.PipelineId, j.PipelineVersionId, "")
		own("RecurringRun", j.UUID, j.ExperimentId, j.PipelineId, j.PipelineVersionId, "")
	}
	for _, v := range b.Versions {
		own("PipelineVersion", v.UUID, "", v.PipelineId, "", "")
	}
	seenRefs := map[string]bool{}
	for _, ref := range b.References {
		key := ref.ResourceType + "\x00" + ref.ResourceUUID + "\x00" + ref.ReferenceType
		if seenRefs[key] || !sets[kinds[ref.ResourceType]][ref.ResourceUUID] || !model.ValidateResourceReferenceRelationship(model.ResourceType(ref.ResourceType), model.ResourceType(ref.ReferenceType), model.Relationship(ref.Relationship)) {
			return invalid("legacy reference has invalid ownership or relationship")
		}
		seenRefs[key] = true
		if targets := expected[ref.ResourceType+"\x00"+ref.ResourceUUID]; targets != nil && ref.ReferenceType != "Namespace" && targets[ref.ReferenceType] != ref.ReferenceUUID {
			return invalid("legacy reference disagrees with the owning record")
		}
		if ref.ReferenceType == "Namespace" {
			if ref.ReferenceUUID != namespace && ref.ReferenceUUID != runtimeNamespace {
				return invalid("legacy reference crosses namespaces")
			}
		} else if !sets[kinds[ref.ReferenceType]][ref.ReferenceUUID] {
			return invalid("legacy reference target is outside archive")
		}
	}
	return nil
}

// Common catalog and run fields retain their released JSON names and meanings.
// The frozen input DTO, followed by explicit cleanup below, makes this bridge
// independent of future additions to the destination's model structs.
func legacyModel(from, to any) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, to)
}

func convertLegacyArchive(old *legacyArchive) (*NamespaceBundle, error) {
	b := &NamespaceBundle{Bundle: Bundle{Format: TransferFormat, Source: old.Source}, Namespace: old.Namespace, RuntimeNamespace: old.RuntimeNamespace, RuntimeParameters: &transfer.RuntimeParameters{Runs: map[string]string{}, Schedules: map[string]string{}}}
	for _, x := range old.Experiments {
		var v model.Experiment
		if err := legacyModel(x, &v); err != nil {
			return nil, err
		}
		b.Experiments = append(b.Experiments, v)
	}
	for _, x := range old.Pipelines {
		var v model.Pipeline
		if err := legacyModel(x, &v); err != nil {
			return nil, err
		}
		v.DefaultVersionId = ""
		b.Pipelines = append(b.Pipelines, v)
		for key, value := range x.Tags {
			b.PipelineTags = append(b.PipelineTags, model.PipelineTag{PipelineID: x.UUID, TagKey: key, TagValue: value})
		}
	}
	for _, x := range old.Versions {
		var v model.PipelineVersion
		if err := legacyModel(x, &v); err != nil {
			return nil, err
		}
		b.Versions = append(b.Versions, v)
		for key, value := range x.Tags {
			b.VersionTags = append(b.VersionTags, model.PipelineVersionTag{PipelineVersionID: x.UUID, TagKey: key, TagValue: value})
		}
	}
	sort.Slice(b.PipelineTags, func(i, j int) bool {
		a, z := b.PipelineTags[i], b.PipelineTags[j]
		return a.PipelineID < z.PipelineID || (a.PipelineID == z.PipelineID && a.TagKey < z.TagKey)
	})
	sort.Slice(b.VersionTags, func(i, j int) bool {
		a, z := b.VersionTags[i], b.VersionTags[j]
		return a.PipelineVersionID < z.PipelineVersionID || (a.PipelineVersionID == z.PipelineVersionID && a.TagKey < z.TagKey)
	})
	resolved := map[string]bool{}
	for i := range b.Pipelines {
		p := &b.Pipelines[i]
		pin := old.CatalogDefaults[p.UUID]
		var selected *model.PipelineVersion
		matches := 0
		for j := range b.Versions {
			v := &b.Versions[j]
			if v.PipelineId != p.UUID {
				continue
			}
			if pin != "" {
				if v.Name == pin {
					selected = v
					matches++
				}
			} else if selected == nil || v.CreatedAtInSec > selected.CreatedAtInSec || (v.CreatedAtInSec == selected.CreatedAtInSec && v.UUID > selected.UUID) {
				selected = v
			}
		}
		if pin != "" && matches != 1 {
			return nil, util.NewInvalidInputError("legacy catalog default must resolve to exactly one archived version")
		}
		if selected != nil {
			p.DefaultVersionId = selected.UUID
		}
		resolved[p.UUID] = true
	}
	for id, pin := range old.CatalogDefaults {
		if !resolved[id] || pin == "" {
			return nil, util.NewInvalidInputError("legacy catalog default has a missing parent or version name")
		}
	}
	for _, x := range old.Schedules {
		var v model.Job
		if err := legacyModel(x, &v); err != nil {
			return nil, err
		}
		v.Namespace = old.RuntimeNamespace
		v.ResourceReferences = nil
		v.RuntimeConfig.Parameters = model.LargeText(old.RuntimeParameters.Schedules[x.UUID])
		b.Schedules = append(b.Schedules, v)
		b.RuntimeParameters.Schedules[v.UUID] = string(v.RuntimeConfig.Parameters)
	}
	for _, x := range old.Runs {
		var e Entry
		r := x.Run
		// Keep the canonical persisted state history rather than duplicate hydrated records.
		if len(r.StateHistory) > 0 {
			historyJSON, err := json.Marshal(r.StateHistory)
			if err != nil {
				return nil, err
			}
			if r.StateHistoryString != "" && !legacyEqualJSON(r.StateHistoryString, string(historyJSON)) {
				return nil, util.NewInvalidInputError("legacy run state history disagrees with its persisted representation")
			}
			r.StateHistoryString = string(historyJSON)
		}
		r.StateHistory = nil
		r.TaskDetails = nil
		r.Metrics = nil
		r.ResourceReferences = nil
		if err := legacyModel(r, &e.Run); err != nil {
			return nil, err
		}
		e.Run.RuntimeConfig.Parameters = model.LargeText(old.RuntimeParameters.Runs[r.UUID])
		e.Run.PipelineContextId = 0
		e.Run.PipelineRunContextId = 0
		e.Run.Namespace = old.RuntimeNamespace
		for _, m := range x.Metrics {
			e.Metrics = append(e.Metrics, model.RunMetricV1{RunUUID: m.RunUUID, NodeID: m.NodeID, Name: m.Name, NumberValue: m.NumberValue, Format: m.Format, Payload: model.LargeText(m.Payload)})
		}
		b.Entries = append(b.Entries, e)
		b.RuntimeParameters.Runs[e.Run.UUID] = string(e.Run.RuntimeConfig.Parameters)
	}
	return b, nil
}

func legacyEqualJSON(a, b string) bool {
	var x, y any
	da := json.NewDecoder(strings.NewReader(a))
	da.UseNumber()
	db := json.NewDecoder(strings.NewReader(b))
	db.UseNumber()
	return da.Decode(&x) == nil && da.Decode(new(any)) == io.EOF && db.Decode(&y) == nil && db.Decode(new(any)) == io.EOF && reflect.DeepEqual(x, y)
}

// Each parent chain is visited once, including adversarial deep task trees.
func validateLegacyTaskParents(tasks map[string]legacyTask) error {
	state := map[string]uint8{}
	for id := range tasks {
		var path []string
		for id != "" && state[id] == 0 {
			task, ok := tasks[id]
			if !ok {
				return util.NewInvalidInputError("legacy task parent is missing")
			}
			state[id] = 1
			path = append(path, id)
			id = task.ParentTaskId
		}
		if id != "" && state[id] == 1 {
			return util.NewInvalidInputError("legacy task parent graph contains a cycle")
		}
		for _, id := range path {
			state[id] = 2
		}
	}
	return nil
}

func validateLegacyRunStateHistory(data []byte) error {
	var statuses []*model.RuntimeStatus
	if err := decodeArchiveJSON(data, &statuses); err != nil {
		return util.NewInvalidInputError("invalid legacy run state history: %v", err)
	}
	for _, state := range statuses {
		if state == nil {
			return util.NewInvalidInputError("legacy run state history cannot contain null entries")
		}
	}
	return nil
}
