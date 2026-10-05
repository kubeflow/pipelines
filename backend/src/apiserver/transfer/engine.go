// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package transfer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const archiveFormat = "kfp-namespace-transfer-mlmd-2.18/v1"
const maxRecords = 100000

// Catalog stages Kubernetes-backed catalog records; SQL catalogs use the transaction.
type Catalog interface {
	Export(context.Context, string) ([]model.Pipeline, []model.PipelineVersion, error)
	Prepare(context.Context, string, string, []model.Pipeline, []model.PipelineVersion, bool) (map[string]string, map[string]string, error)
}

// Schedules creates or resumes a disabled, provenance-owned ScheduledWorkflow.
type Schedules interface {
	Prepare(context.Context, string, *model.Job, string, []byte, bool) (*model.Job, error)
}

// Engine publishes SQL records only after external stores have been staged.
type Engine struct {
	DB                   *gorm.DB
	RuntimeNamespace     string
	LoadPipelineSpec     func(context.Context, *model.PipelineVersion) ([]byte, error)
	ValidatePipelineSpec func([]byte) error
	Catalog              Catalog
	Schedules            Schedules
	Metadata             *Metadata
}

type Bundle struct {
	Format           string                    `json:"format"`
	Source           string                    `json:"source"`
	Namespace        string                    `json:"namespace"`
	RuntimeNamespace string                    `json:"runtime_namespace"`
	Schema           string                    `json:"schema"`
	Experiments      []model.Experiment        `json:"experiments"`
	Pipelines        []model.Pipeline          `json:"pipelines"`
	Versions         []model.PipelineVersion   `json:"pipeline_versions"`
	Schedules        []model.Job               `json:"schedules"`
	Runs             []RunHistory              `json:"runs"`
	References       []model.ResourceReference `json:"references"`
	Metadata         Graph                     `json:"metadata"`
	Digest           string                    `json:"digest"`
}

type RunHistory struct {
	Run     model.Run         `json:"run"`
	Tasks   []model.Task      `json:"tasks"`
	Metrics []model.RunMetric `json:"metrics"`
}

func hash(value any) string {
	b, _ := json.Marshal(value)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func equal(column string, value any) clause.Expression {
	return clause.Eq{Column: clause.Column{Name: column}, Value: value}
}

func schemaSignature(db *gorm.DB) (string, error) {
	shape := []string{db.Name()}
	for _, row := range []any{&model.Experiment{}, &model.Pipeline{}, &model.PipelineVersion{}, &model.PipelineTag{}, &model.PipelineVersionTag{}, &model.Run{}, &model.Task{}, &model.Job{}, &model.RunMetric{}, &model.ResourceReference{}} {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(row); err != nil {
			return "", err
		}
		cols, err := db.Migrator().ColumnTypes(row)
		if err != nil {
			return "", err
		}
		var names []string
		for _, col := range cols {
			length, lok := col.Length()
			nullable, nok := col.Nullable()
			names = append(names, fmt.Sprintf("%s.%s:%s:%d:%t:%t:%t", stmt.Table, col.Name(), col.DatabaseTypeName(), length, lok, nullable, nok))
		}
		sort.Strings(names)
		shape = append(shape, names...)
	}
	return hash(shape), nil
}

type exportBudget struct{ records, bytes int }

func collect[T any](query *gorm.DB, out *[]T, budget *exportBudget) error {
	rows, err := query.Model(new(T)).Limit(maxRecords + 1).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var row T
		if err := query.ScanRows(rows, &row); err != nil {
			return err
		}
		data, err := json.Marshal(row)
		if err != nil {
			return err
		}
		budget.records++
		budget.bytes += len(data)
		if budget.records > maxRecords || budget.bytes > MaxArchiveBytes {
			return util.NewInvalidInputError("Namespace exceeds transfer limit; select a smaller completion-time interval")
		}
		*out = append(*out, row)
	}
	return rows.Err()
}

func (e *Engine) sourceID(ctx context.Context) (string, error) {
	id := model.TransferIdentity{ID: 1, UUID: uuid.NewString()}
	if err := e.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&id).Error; err != nil {
		return "", err
	}
	if err := e.DB.WithContext(ctx).First(&id, 1).Error; err != nil {
		return "", err
	}
	return id.UUID, nil
}

func terminal(r model.Run) bool {
	state := r.State
	if state == "" {
		state = model.RuntimeState(r.Conditions)
	}
	switch state.ToV2() {
	case model.RuntimeStateSucceeded, model.RuntimeStateFailed, model.RuntimeStateCanceled, model.RuntimeStateSkipped:
		return r.FinishedAtInSec > 0
	default:
		return false
	}
}

// Export takes a consistent SQL snapshot. Completed runs and catalog definitions must
// remain unchanged while their independently stored MLMD and catalog data is read.
func (e *Engine) Export(ctx context.Context, namespace string, opts ExportOptions) ([]byte, error) {
	if opts.CompletedAfter < 0 || opts.CompletedBefore < 0 || (opts.CompletedBefore != 0 && opts.CompletedAfter >= opts.CompletedBefore) {
		return nil, util.NewInvalidInputError("specify a namespace and a valid completion-time interval")
	}
	source, err := e.sourceID(ctx)
	if err != nil {
		return nil, err
	}
	runtimeNamespace := namespace
	if runtimeNamespace == "" {
		runtimeNamespace = e.RuntimeNamespace
	}
	b := Bundle{Format: archiveFormat, Source: source, Namespace: namespace, RuntimeNamespace: runtimeNamespace, Metadata: Graph{}}
	b.Schema, err = schemaSignature(e.DB)
	if err != nil {
		return nil, err
	}
	budget := &exportBudget{}
	err = e.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := collect(tx.Where(equal("Namespace", namespace)), &b.Experiments, budget); err != nil {
			return err
		}
		var experimentIDs []string
		for _, exp := range b.Experiments {
			experimentIDs = append(experimentIDs, exp.UUID)
		}
		if err := collect(tx.Where(map[string]any{"ExperimentUUID": experimentIDs}), &b.Schedules, budget); err != nil {
			return err
		}
		var runs []model.Run
		q := tx.Where(map[string]any{"ExperimentUUID": experimentIDs}).Where(clause.Gte{Column: clause.Column{Name: "FinishedAtInSec"}, Value: opts.CompletedAfter})
		if opts.CompletedBefore > 0 {
			q = q.Where(clause.Lt{Column: clause.Column{Name: "FinishedAtInSec"}, Value: opts.CompletedBefore})
		}
		if err := collect(q, &runs, budget); err != nil {
			return err
		}
		if len(runs) > maxRecords {
			return util.NewInvalidInputError("Too many runs; select a smaller completion-time interval")
		}
		for _, run := range runs {
			if !terminal(run) {
				continue
			}
			run.ImportedFrom = nil
			run.ImportDigest = nil
			h := RunHistory{Run: run}
			if err := collect(tx.Where(equal("RunUUID", run.UUID)), &h.Tasks, budget); err != nil {
				return err
			}
			if err := collect(tx.Where(equal("RunUUID", run.UUID)), &h.Metrics, budget); err != nil {
				return err
			}
			b.Runs = append(b.Runs, h)
		}
		if e.Catalog == nil {
			catalogQuery := tx.Where(equal("Namespace", namespace))
			if namespace == "" {
				catalogQuery = tx.Where(map[string]any{"Namespace": []string{"", model.NoNamespace}})
			}
			if err := collect(catalogQuery, &b.Pipelines, budget); err != nil {
				return err
			}
			for i := range b.Pipelines {
				p := &b.Pipelines[i]
				if namespace == "" && p.Namespace == model.NoNamespace {
					p.Namespace = ""
				}
				var tags []model.PipelineTag
				if err := collect(tx.Where(equal("PipelineId", p.UUID)), &tags, budget); err != nil {
					return err
				}
				p.Tags = map[string]string{}
				for _, tag := range tags {
					p.Tags[tag.TagKey] = tag.TagValue
				}
				var versions []model.PipelineVersion
				if err := collect(tx.Where(equal("PipelineId", p.UUID)), &versions, budget); err != nil {
					return err
				}
				for j := range versions {
					var tags []model.PipelineVersionTag
					if err := collect(tx.Where(equal("PipelineVersionId", versions[j].UUID)), &tags, budget); err != nil {
						return err
					}
					versions[j].Tags = map[string]string{}
					for _, tag := range tags {
						versions[j].Tags[tag.TagKey] = tag.TagValue
					}
				}
				b.Versions = append(b.Versions, versions...)
			}
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	if e.Catalog != nil {
		b.Pipelines, b.Versions, err = e.Catalog.Export(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}
	owners := map[string]bool{}
	for _, p := range b.Pipelines {
		owners[p.UUID] = true
	}
	for _, v := range b.Versions {
		owners[v.UUID] = true
	}
	for _, x := range b.Experiments {
		owners[x.UUID] = true
	}
	for _, j := range b.Schedules {
		owners[j.UUID] = true
	}
	for _, r := range b.Runs {
		owners[r.Run.UUID] = true
	}
	var ownerIDs []string
	for id := range owners {
		ownerIDs = append(ownerIDs, id)
	}
	for start := 0; start < len(ownerIDs); start += 500 {
		end := start + 500
		if end > len(ownerIDs) {
			end = len(ownerIDs)
		}
		var refs []model.ResourceReference
		if err := collect(e.DB.WithContext(ctx).Where(map[string]any{"ResourceUUID": ownerIDs[start:end]}), &refs, budget); err != nil {
			return nil, err
		}
		b.References = append(b.References, refs...)
	}
	for i := range b.Versions {
		if b.Versions[i].PipelineSpec == "" {
			if e.LoadPipelineSpec == nil {
				return nil, util.NewInvalidInputError("Pipeline version has no embedded definition")
			}
			spec, err := e.LoadPipelineSpec(ctx, &b.Versions[i])
			if err != nil {
				return nil, err
			}
			budget.bytes += len(spec)
			if budget.bytes > MaxArchiveBytes {
				return nil, util.NewInvalidInputError("Catalog definitions exceed transfer size limit")
			}
			b.Versions[i].PipelineSpec = model.LargeText(spec)
		}
		// Archive definitions are self-contained; object-store identity stays at the source.
		b.Versions[i].PipelineSpecURI = ""
	}
	normalizeDeletedCatalog(&b)

	if e.Metadata != nil {
		e.Metadata.Namespace = namespace
		b.Metadata, err = e.Metadata.Export(ctx, b.Runs, func(id string) error {
			var run model.Run
			if err := e.DB.WithContext(ctx).Where(equal("UUID", id)).First(&run).Error; err != nil {
				return util.NewInvalidInputError("MLMD provenance run %s is unavailable; restore its producer history before exporting", id)
			}
			var experiment model.Experiment
			if err := e.DB.WithContext(ctx).Where(equal("UUID", run.ExperimentId)).First(&experiment).Error; err != nil {
				return util.NewInvalidInputError("MLMD provenance experiment is unavailable")
			}
			if experiment.Namespace != namespace {
				return util.NewInvalidInputError("MLMD ancestry crosses experiment namespaces")
			}
			if run.Namespace != b.RuntimeNamespace && run.Namespace != namespace {
				return util.NewInvalidInputError("MLMD ancestry crosses namespaces; export this history separately after removing cross-namespace references")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		for _, r := range b.Runs {
			if r.Run.PipelineRuntimeManifest != "" || r.Run.PipelineContextId != 0 || r.Run.PipelineRunContextId != 0 || len(r.Tasks) > 0 {
				return nil, util.NewInvalidInputError("metadata service is required to export complete run history")
			}
		}
	}
	if err := validateBundle(&b); err != nil {
		return nil, err
	}
	b.Digest = hash(b)
	data, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxArchiveBytes {
		return nil, util.NewInvalidInputError("transfer exceeds 256 MiB; select a smaller completion-time interval")
	}
	return data, nil
}

// Deleted catalog definitions do not prevent historical run transfer. Keep the
// embedded/runtime specs and remove links that would point outside this archive.
func normalizeDeletedCatalog(b *Bundle) {
	pipelines, versions := map[string]bool{}, map[string]bool{}
	for _, p := range b.Pipelines {
		pipelines[p.UUID] = true
	}
	for _, v := range b.Versions {
		versions[v.UUID] = true
	}
	clear := func(p *model.PipelineSpec) {
		if !pipelines[p.PipelineId] {
			p.PipelineId = ""
		}
		if !versions[p.PipelineVersionId] {
			p.PipelineVersionId = ""
		}
		if p.PipelineId == "" && p.PipelineVersionId == "" {
			p.PipelineName = ""
		}
	}
	for i := range b.Runs {
		clear(&b.Runs[i].Run.PipelineSpec)
	}
	for i := range b.Schedules {
		if b.Schedules[i].PipelineSpecManifest != "" || b.Schedules[i].WorkflowSpecManifest != "" {
			clear(&b.Schedules[i].PipelineSpec)
		}
	}
	refs := b.References[:0]
	for _, r := range b.References {
		if r.ReferenceType == model.PipelineResourceType && !pipelines[r.ReferenceUUID] {
			continue
		}
		if r.ReferenceType == model.PipelineVersionResourceType && !versions[r.ReferenceUUID] {
			continue
		}
		refs = append(refs, r)
	}
	b.References = refs
}

func validateID(id string) bool {
	return id != "" && utf8.ValidString(id) && len(id) <= 191 && !strings.ContainsRune(id, 0)
}

func validateBundle(b *Bundle) error {
	if b.Format != archiveFormat {
		return util.NewInvalidInputError("archive is not a compatible release-2.18 namespace transfer")
	}
	if _, err := uuid.Parse(b.Source); err != nil {
		return util.NewInvalidInputError("invalid source installation identity")
	}
	if b.Namespace != "" && b.RuntimeNamespace != b.Namespace {
		return util.NewInvalidInputError("Runtime namespace must equal the archive namespace")
	}
	if len(b.Namespace) > 63 || len(b.RuntimeNamespace) > 63 {
		return util.NewInvalidInputError("invalid archive namespace")
	}
	sets := map[string]map[string]bool{}
	count := 0
	add := func(kind, id string) error {
		if !validateID(id) {
			return fmt.Errorf("invalid %s ID", kind)
		}
		if sets[kind] == nil {
			sets[kind] = map[string]bool{}
		}
		if sets[kind][id] {
			return fmt.Errorf("duplicate %s ID", kind)
		}
		sets[kind][id] = true
		count++
		return nil
	}
	for _, x := range b.Experiments {
		if x.Namespace != b.Namespace {
			return util.NewInvalidInputError("experiment namespace differs from archive")
		}
		if err := add("experiment", x.UUID); err != nil {
			return err
		}
	}
	for _, x := range b.Pipelines {
		if x.Namespace != b.Namespace || x.Status != model.PipelineReady {
			return util.NewInvalidInputError("pipeline is not ready in the archive namespace")
		}
		if err := add("pipeline", x.UUID); err != nil {
			return err
		}
	}
	for _, x := range b.Versions {
		if x.PipelineSpec == "" || x.PipelineSpecURI != "" {
			return util.NewInvalidInputError("Pipeline versions require an inline definition and no source URI")
		}
		if !sets["pipeline"][x.PipelineId] || x.Status != model.PipelineVersionReady {
			return util.NewInvalidInputError("pipeline version has a missing parent or is not ready")
		}
		if err := add("version", x.UUID); err != nil {
			return err
		}
		if x.Pipeline.UUID != "" {
			return util.NewInvalidInputError("nested pipeline records are not accepted")
		}
	}
	for _, x := range b.Schedules {
		if (x.Namespace != b.Namespace && x.Namespace != b.RuntimeNamespace) || !sets["experiment"][x.ExperimentId] {
			return util.NewInvalidInputError("schedule has a missing experiment or wrong namespace")
		}
		if err := add("schedule", x.UUID); err != nil {
			return err
		}
	}
	checkSpec := func(p model.PipelineSpec) error {
		if p.PipelineId != "" && !sets["pipeline"][p.PipelineId] {
			return util.NewInvalidInputError("pipeline reference is outside archive namespace/catalog")
		}
		if p.PipelineVersionId != "" && !sets["version"][p.PipelineVersionId] {
			return util.NewInvalidInputError("pipeline version reference is outside archive catalog")
		}
		for _, v := range b.Versions {
			if v.UUID == p.PipelineVersionId && p.PipelineId != "" && v.PipelineId != p.PipelineId {
				return util.NewInvalidInputError("pipeline version belongs to a different pipeline")
			}
		}
		return nil
	}
	for _, x := range b.Schedules {
		if err := checkSpec(x.PipelineSpec); err != nil {
			return err
		}
	}
	for _, h := range b.Runs {
		r := h.Run
		if (r.Namespace != b.Namespace && r.Namespace != b.RuntimeNamespace) || !sets["experiment"][r.ExperimentId] || !terminal(r) || r.ImportedFrom != nil || r.ImportDigest != nil {
			return util.NewInvalidInputError("run must be completed native history in the archive namespace")
		}
		if err := add("run", r.UUID); err != nil {
			return err
		}
		if err := checkSpec(r.PipelineSpec); err != nil {
			return err
		}
		if r.RecurringRunId != "" && !sets["schedule"][r.RecurringRunId] {
			return util.NewInvalidInputError("run references a missing schedule")
		}
		tasks := map[string]bool{}
		edges := [][2]string{}
		for _, t := range h.Tasks {
			if t.RunID != r.UUID || t.Namespace != r.Namespace || t.Run.UUID != "" {
				return util.NewInvalidInputError("task belongs to a different run or namespace")
			}
			if err := add("task", t.UUID); err != nil {
				return err
			}
			tasks[t.UUID] = true
		}
		for _, t := range h.Tasks {
			if t.ParentTaskId != "" {
				if !tasks[t.ParentTaskId] {
					return util.NewInvalidInputError("task parent is outside its run")
				}
				edges = append(edges, [2]string{t.ParentTaskId, t.UUID})
			}
			if t.Payload != "" {
				var payload model.Task
				if err := json.Unmarshal([]byte(t.Payload), &payload); err != nil {
					return util.NewInvalidInputError("Invalid task payload: %v", err)
				}
				if payload.UUID != t.UUID || payload.RunID != t.RunID || payload.Namespace != t.Namespace {
					return util.NewInvalidInputError("task payload identity differs from task")
				}
			}
		}
		if err := acyclic(edges); err != nil {
			return err
		}
		metricKeys := map[string]bool{}
		for _, m := range h.Metrics {
			key := m.NodeID + "\x00" + m.Name
			if m.RunUUID != r.UUID || metricKeys[key] {
				return util.NewInvalidInputError("invalid or duplicate run metric")
			}
			metricKeys[key] = true
		}
	}
	if count > maxRecords {
		return util.NewInvalidInputError("archive exceeds 100000 SQL records")
	}
	kinds := map[model.ResourceType]string{model.ExperimentResourceType: "experiment", model.PipelineResourceType: "pipeline", model.PipelineVersionResourceType: "version", model.JobResourceType: "schedule", model.RecurringRunResourceType: "schedule", model.RunResourceType: "run"}
	seenRefs := map[string]bool{}
	for _, ref := range b.References {
		if !sets[kinds[ref.ResourceType]][ref.ResourceUUID] || !model.ValidateResourceReferenceRelationship(ref.ResourceType, ref.ReferenceType, ref.Relationship) {
			return util.NewInvalidInputError("resource reference owner or relationship is invalid")
		}
		if ref.ReferenceType == model.NamespaceResourceType {
			if ref.ReferenceUUID != b.Namespace && ref.ReferenceUUID != b.RuntimeNamespace {
				return util.NewInvalidInputError("resource reference crosses namespaces")
			}
		} else if !sets[kinds[ref.ReferenceType]][ref.ReferenceUUID] {
			return util.NewInvalidInputError("resource reference target is outside archive")
		}
		key := hash([]any{ref.ResourceType, ref.ResourceUUID, ref.ReferenceType})
		if seenRefs[key] {
			return util.NewInvalidInputError("duplicate resource reference")
		}
		seenRefs[key] = true
	}
	return validateGraph(b.Metadata, b.Runs)
}

func receiptFor(b *Bundle, kind, id string, value any, prefix string) model.TransferReceipt {
	return model.TransferReceipt{Key: hash([]string{b.Source, b.Namespace, kind, id}), Source: b.Source, Namespace: b.Namespace, Kind: kind, SourceID: id, TargetID: id, Digest: hash(value), NamePrefix: prefix}
}

func pipelineDigest(x model.Pipeline) any { x.DefaultVersionId = ""; return x }

func scheduleDigest(x model.Job) any { x.UpdatedAtInSec = 0; x.Conditions = ""; return x }

func experimentDigest(x model.Experiment) any { x.LastRunCreatedAtInSec = 0; return x }

// Import validates all identities before staging and rechecks them in the publishing transaction.
func (e *Engine) Import(ctx context.Context, namespace string, data []byte, opts ImportOptions) (Summary, error) {
	result := Summary{DryRun: opts.DryRun, Warnings: []string{"Artifact files are not copied; retain access to the shared bucket.", "Schedules are imported disabled with catch-up disabled. Review them before enabling.", "New SQL pipeline versions follow normal latest-version selection; existing Kubernetes defaults remain unchanged.", "Failed imports may leave staged catalog, schedule, or metadata records; retry the same archive."}}
	if len(data) > MaxArchiveBytes {
		return result, util.NewInvalidInputError("archive exceeds 256 MiB")
	}
	var b Bundle
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(&b); err != nil {
		return result, util.NewInvalidInputError("Invalid archive: %v", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return result, util.NewInvalidInputError("archive must contain exactly one JSON value")
	}
	digest := b.Digest
	b.Digest = ""
	if digest == "" || hash(b) != digest {
		return result, util.NewInvalidInputError("archive digest does not match its contents")
	}
	if namespace != b.Namespace || (namespace == "" && b.RuntimeNamespace != e.RuntimeNamespace) {
		return result, util.NewInvalidInputError("destination namespace must match archive namespace")
	}
	if err := validateBundle(&b); err != nil {
		return result, err
	}
	schema, err := schemaSignature(e.DB)
	if err != nil {
		return result, err
	}
	if schema != b.Schema {
		return result, util.NewInvalidInputError("database schema differs; use matching release versions")
	}
	if len(opts.NamePrefix) > 64 || !utf8.ValidString(opts.NamePrefix) {
		return result, util.NewInvalidInputError("name prefix must contain at most 64 UTF-8 bytes")
	}
	result.Counts = Counts{Experiments: len(b.Experiments), Pipelines: len(b.Pipelines), PipelineVersions: len(b.Versions), Runs: len(b.Runs), Schedules: len(b.Schedules)}
	receipts := map[string]model.TransferReceipt{}
	skipped := map[string]bool{}
	add := func(kind, id string, value any) {
		r := receiptFor(&b, kind, id, value, opts.NamePrefix)
		receipts[kind+":"+id] = r
	}
	for _, x := range b.Experiments {
		add("experiment", x.UUID, experimentDigest(x))
	}
	for _, x := range b.Pipelines {
		add("pipeline", x.UUID, pipelineDigest(x))
	}
	for _, x := range b.Versions {
		add("version", x.UUID, x)
	}
	for _, x := range b.Schedules {
		add("schedule", x.UUID, scheduleDigest(x))
	}
	for _, x := range b.Runs {
		add("run", x.Run.UUID, x)
	}
	for i := range b.Experiments {
		b.Experiments[i].Name = opts.NamePrefix + b.Experiments[i].Name
	}
	for i := range b.Pipelines {
		b.Pipelines[i].Name = opts.NamePrefix + b.Pipelines[i].Name
	}
	for i := range b.Schedules {
		b.Schedules[i].DisplayName = opts.NamePrefix + b.Schedules[i].DisplayName
	}
	preflight := func(db *gorm.DB) error {
		for key, want := range receipts {
			var old model.TransferReceipt
			err := db.Where(equal("Key", want.Key)).First(&old).Error
			if err == nil {
				if old.Source != want.Source || old.Namespace != namespace || old.Digest != want.Digest || old.NamePrefix != opts.NamePrefix {
					return util.NewInvalidInputError("resource conflicts with an earlier transfer; use its original archive and prefix")
				}
				receipts[key] = old
				skipped[key] = true
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		for i := range b.Experiments {
			if err := checkNew(db, &b.Experiments[i], skipped["experiment:"+b.Experiments[i].UUID], true); err != nil {
				return err
			}
		}
		if e.Catalog == nil {
			for i := range b.Pipelines {
				if err := checkNew(db, &b.Pipelines[i], skipped["pipeline:"+b.Pipelines[i].UUID], true); err != nil {
					return err
				}
			}
			for i := range b.Versions {
				if err := checkNew(db, &b.Versions[i], skipped["version:"+b.Versions[i].UUID], true); err != nil {
					return err
				}
			}
		}
		for i := range b.Runs {
			h := &b.Runs[i]
			if err := checkNew(db, &h.Run, skipped["run:"+h.Run.UUID], false); err != nil {
				return err
			}
			if !skipped["run:"+h.Run.UUID] {
				for j := range h.Tasks {
					if err := checkNew(db, &h.Tasks[j], false, false); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	for _, p := range b.Pipelines {
		if err := model.ValidateTags(p.Tags); err != nil {
			return result, err
		}
	}
	for _, v := range b.Versions {
		if e.ValidatePipelineSpec != nil {
			if err := e.ValidatePipelineSpec([]byte(v.PipelineSpec)); err != nil {
				return result, err
			}
		}
		if err := model.ValidateTags(v.Tags); err != nil {
			return result, err
		}
	}
	for i := range b.Schedules {
		stmt := &gorm.Statement{DB: e.DB}
		if err := stmt.Parse(&b.Schedules[i]); err != nil {
			return result, err
		}
		if err := validateRow(stmt, reflect.ValueOf(b.Schedules[i])); err != nil {
			return result, err
		}
	}
	if err := preflight(e.DB.WithContext(ctx)); err != nil {
		return result, err
	}
	for key := range receipts {
		if skipped[key] {
			result.Skipped++
		} else {
			result.Imported++
		}
	}
	if e.Metadata == nil && len(b.Metadata) > 0 {
		return result, util.NewInvalidInputError("metadata service is required for this archive")
	}
	if e.Metadata != nil {
		e.Metadata.Namespace = namespace
		if err := e.Metadata.Preflight(ctx, b.Metadata, b.Source); err != nil {
			return result, err
		}
	}
	if result.Imported == 0 {
		return result, nil
	}
	pipelineMap, versionMap := map[string]string{}, map[string]string{}
	for _, p := range b.Pipelines {
		pipelineMap[p.UUID] = p.UUID
	}
	for _, v := range b.Versions {
		versionMap[v.UUID] = v.UUID
	}
	if e.Catalog != nil {
		pipelineMap, versionMap, err = e.Catalog.Prepare(ctx, b.Source, namespace, b.Pipelines, b.Versions, true)
		if err != nil {
			return result, err
		}
	}
	scheduleSpecs := map[string][]byte{}
	for _, j := range b.Schedules {
		manifest := j.PipelineSpecManifest
		if manifest == "" {
			manifest = j.WorkflowSpecManifest
		}
		var selected *model.PipelineVersion
		for k := range b.Versions {
			v := &b.Versions[k]
			if j.PipelineVersionId != "" {
				if v.UUID == j.PipelineVersionId {
					selected = v
					break
				}
			} else if j.PipelineId != "" && v.PipelineId == j.PipelineId {
				if selected == nil || v.CreatedAtInSec > selected.CreatedAtInSec || (v.CreatedAtInSec == selected.CreatedAtInSec && v.UUID > selected.UUID) {
					selected = v
				}
			}
		}
		if selected != nil {
			manifest = selected.PipelineSpec
		}
		if manifest == "" {
			return result, util.NewInvalidInputError("Schedule has no complete pipeline definition")
		}
		scheduleSpecs[j.UUID] = []byte(manifest)
	}
	for i := range b.Schedules {
		if skipped["schedule:"+b.Schedules[i].UUID] {
			continue
		}
		if e.Schedules == nil {
			return result, util.NewInvalidInputError("schedule service unavailable")
		}
		if _, err := e.Schedules.Prepare(ctx, b.Source, &b.Schedules[i], receipts["schedule:"+b.Schedules[i].UUID].Digest, scheduleSpecs[b.Schedules[i].UUID], true); err != nil {
			return result, err
		}
	}
	if opts.DryRun {
		return result, nil
	}
	if e.Catalog != nil {
		pipelineMap, versionMap, err = e.Catalog.Prepare(ctx, b.Source, namespace, b.Pipelines, b.Versions, false)
		if err != nil {
			return result, err
		}
	}
	remapSpec := func(p *model.PipelineSpec) {
		if p.PipelineId != "" {
			p.PipelineId = pipelineMap[p.PipelineId]
		}
		if p.PipelineVersionId != "" {
			p.PipelineVersionId = versionMap[p.PipelineVersionId]
		}
		p.PipelineName = ""
	}
	scheduleMap := map[string]string{}
	for i := range b.Schedules {
		j := &b.Schedules[i]
		oldID := j.UUID
		rec := receipts["schedule:"+oldID]
		if skipped["schedule:"+oldID] {
			scheduleMap[oldID] = rec.TargetID
			*j = model.Job{}
			if err := e.DB.WithContext(ctx).Where(equal("UUID", rec.TargetID)).First(j).Error; err != nil {
				return result, util.NewInvalidInputError("Previously imported schedule was deleted; restore it before retrying")
			}
			if j.Namespace != b.Namespace && j.Namespace != b.RuntimeNamespace {
				return result, util.NewInvalidInputError("Schedule receipt target belongs to another namespace")
			}
			continue
		}
		remapSpec(&j.PipelineSpec)
		j.Enabled = false
		j.NoCatchup = true
		staged, err := e.Schedules.Prepare(ctx, b.Source, j, rec.Digest, scheduleSpecs[oldID], false)
		if err != nil {
			return result, err
		}
		*j = *staged
		scheduleMap[oldID] = j.UUID
		rec.TargetID = j.UUID
		receipts["schedule:"+oldID] = rec
	}
	mapping := IDMapping{}
	if e.Metadata != nil {
		mapping, err = e.Metadata.Stage(ctx, b.Metadata, b.Source)
		if err != nil {
			return result, err
		}
	}
	for i := range b.Runs {
		h := &b.Runs[i]
		r := &h.Run
		remapSpec(&r.PipelineSpec)
		if r.RecurringRunId != "" {
			r.RecurringRunId = scheduleMap[r.RecurringRunId]
		}
		source := b.Source
		digest := receipts["run:"+r.UUID].Digest
		r.ImportedFrom = &source
		r.ImportDigest = &digest
		r.RetryClaimedAtInSec = 0
		if err := remapMetadata(h, mapping); err != nil {
			return result, err
		}
	}
	for _, kind := range []string{"pipeline", "version"} {
		m := pipelineMap
		if kind == "version" {
			m = versionMap
		}
		for old, target := range m {
			key := kind + ":" + old
			rec := receipts[key]
			rec.TargetID = target
			receipts[key] = rec
		}
	}
	return result, e.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// A concurrent importer can win during external staging. Its receipts are
		// checked again, and conflicting IDs abort the complete SQL publication.
		if err := preflight(tx); err != nil {
			return err
		}
		for i := range b.Experiments {
			if !skipped["experiment:"+b.Experiments[i].UUID] {
				if err := create(tx, &b.Experiments[i]); err != nil {
					return err
				}
			}
		}
		if e.Catalog == nil {
			for i := range b.Pipelines {
				p := &b.Pipelines[i]
				if skipped["pipeline:"+p.UUID] {
					continue
				}
				if err := create(tx, p); err != nil {
					return err
				}
				for name, value := range p.Tags {
					if err := create(tx, &model.PipelineTag{PipelineID: p.UUID, TagKey: name, TagValue: value}); err != nil {
						return err
					}
				}
			}
			for i := range b.Versions {
				v := &b.Versions[i]
				if skipped["version:"+v.UUID] {
					continue
				}
				if err := create(tx, v); err != nil {
					return err
				}
				for name, value := range v.Tags {
					if err := create(tx, &model.PipelineVersionTag{PipelineVersionID: v.UUID, TagKey: name, TagValue: value}); err != nil {
						return err
					}
				}
			}
		}
		for i := range b.Schedules {
			j := &b.Schedules[i]
			var skip bool
			for old, target := range scheduleMap {
				if target == j.UUID {
					skip = skipped["schedule:"+old]
					break
				}
			}
			if !skip {
				if err := create(tx, j); err != nil {
					return err
				}
			}
		}
		for i := range b.Runs {
			h := &b.Runs[i]
			if skipped["run:"+h.Run.UUID] {
				continue
			}
			if err := create(tx, &h.Run); err != nil {
				return err
			}
			for j := range h.Tasks {
				if err := create(tx, &h.Tasks[j]); err != nil {
					return err
				}
			}
			for j := range h.Metrics {
				if err := create(tx, &h.Metrics[j]); err != nil {
					return err
				}
			}
		}
		for _, ref := range b.References {
			kind := referenceKind(ref.ResourceType)
			if skipped[kind+":"+ref.ResourceUUID] {
				continue
			}
			ref.ResourceUUID = mapReference(ref.ResourceType, ref.ResourceUUID, pipelineMap, versionMap, scheduleMap)
			ref.ReferenceUUID = mapReference(ref.ReferenceType, ref.ReferenceUUID, pipelineMap, versionMap, scheduleMap)
			ref.Payload = ""
			payload, _ := json.Marshal(ref)
			ref.Payload = model.LargeText(payload)
			if err := create(tx, &ref); err != nil {
				return err
			}
		}
		for key, rec := range receipts {
			if !skipped[key] {
				if err := create(tx, &rec); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func referenceKind(t model.ResourceType) string {
	switch t {
	case model.ExperimentResourceType:
		return "experiment"
	case model.PipelineResourceType:
		return "pipeline"
	case model.PipelineVersionResourceType:
		return "version"
	case model.JobResourceType, model.RecurringRunResourceType:
		return "schedule"
	case model.RunResourceType:
		return "run"
	}
	return ""
}
func mapReference(t model.ResourceType, id string, p, v, j map[string]string) string {
	switch referenceKind(t) {
	case "pipeline":
		return p[id]
	case "version":
		return v[id]
	case "schedule":
		return j[id]
	}
	return id
}

func checkNew(db *gorm.DB, row any, skip, names bool) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(row); err != nil {
		return err
	}
	rv := reflect.Indirect(reflect.ValueOf(row))
	pk := stmt.Schema.PrioritizedPrimaryField
	if pk == nil {
		return util.NewInvalidInputError("transfer model has no primary key")
	}
	id, _ := pk.ValueOf(context.Background(), rv)
	var count int64
	if err := db.Model(row).Where(equal(pk.DBName, id)).Count(&count).Error; err != nil {
		return err
	}
	if skip {
		if count != 1 {
			return util.NewInvalidInputError("previously imported resource was deleted; restore it before retrying")
		}
		return nil
	}
	if count != 0 {
		return util.NewAlreadyExistError("%s ID %v already exists; existing records are never overwritten", stmt.Table, id)
	}
	if names {
		q := db.Model(row)
		for _, name := range []string{"Namespace", "PipelineId", "Name"} {
			if field := stmt.Schema.FieldsByName[name]; field != nil {
				v, _ := field.ValueOf(context.Background(), rv)
				q = q.Where(equal(field.DBName, v))
			}
		}
		if err := q.Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return util.NewAlreadyExistError("%s name conflicts; choose a name prefix", stmt.Table)
		}
	}
	return validateRow(stmt, rv)
}

func validateRow(stmt *gorm.Statement, rv reflect.Value) error {
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		value, _ := field.ValueOf(context.Background(), rv)
		if text, ok := value.(string); ok && field.Size > 0 && utf8.RuneCountInString(text) > field.Size {
			return util.NewInvalidInputError("%s.%s exceeds its maximum length", stmt.Table, field.DBName)
		}
	}
	return nil
}

func create(db *gorm.DB, row any) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(row); err != nil {
		return err
	}
	rv := reflect.Indirect(reflect.ValueOf(row))
	if err := validateRow(stmt, rv); err != nil {
		return err
	}
	values := map[string]any{}
	for _, name := range stmt.Schema.DBNames {
		field := stmt.Schema.FieldsByDBName[name]
		value, _ := field.ValueOf(context.Background(), rv)
		values[name] = value
	}
	return db.Model(row).Omit(clause.Associations).Create(values).Error
}
