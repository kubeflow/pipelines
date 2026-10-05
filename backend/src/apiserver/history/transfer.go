// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const TransferFormat = "kfp-namespace-transfer/v1"
const MaxTransferRuns = 1000
const MaxTransferObjects = 10000
const MaxTransferRows = 20000

// NamespaceBundle is a logical namespace archive. RuntimeNamespace is used only
// for single-user installations whose experiment namespace is empty.
type NamespaceBundle struct {
	Bundle
	Namespace        string      `json:"namespace"`
	RuntimeNamespace string      `json:"runtime_namespace"`
	Schedules        []model.Job `json:"schedules"`
}

// InstallationID is created once, including under concurrent API replicas.
func InstallationID(ctx context.Context, db *gorm.DB) (string, error) {
	row := model.TransferIdentity{Key: "installation", UUID: uuid.NewString()}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return "", err
	}
	if err := db.WithContext(ctx).Where(clause.Eq{Column: "Key", Value: "installation"}).Take(&row).Error; err != nil {
		return "", err
	}
	return row.UUID, nil
}

// ExportNamespace reads SQL history consistently. Catalog definitions are
// supplied by the configured store, including Kubernetes-backed catalogs.
func ExportNamespace(ctx context.Context, db *gorm.DB, namespace, runtimeNamespace string, opts transfer.ExportOptions, pipelines []model.Pipeline, versions []model.PipelineVersion, normalize ...func(*NamespaceBundle) error) (*NamespaceBundle, error) {
	if opts.CompletedAfter < 0 || opts.CompletedBefore < 0 || (opts.CompletedBefore > 0 && opts.CompletedAfter >= opts.CompletedBefore) {
		return nil, util.NewInvalidInputError("invalid completed-history time window")
	}
	source, err := InstallationID(ctx, db)
	if err != nil {
		return nil, err
	}
	schema, err := schemaSignature(db)
	if err != nil {
		return nil, err
	}
	budget := transfer.NewExportBudget(transfer.MaxArchiveBytes)
	// Reserve the small top-level object and per-entry collection keys separately.
	if err := budget.Reserve(4096); err != nil {
		return nil, err
	}
	for _, p := range pipelines {
		if err := budget.Add(p); err != nil {
			return nil, err
		}
	}
	for _, v := range versions {
		if err := budget.Add(v); err != nil {
			return nil, err
		}
	}
	b := &NamespaceBundle{Bundle: Bundle{Format: TransferFormat, Source: source, Schema: schema, Pipelines: append([]model.Pipeline(nil), pipelines...), Versions: append([]model.PipelineVersion(nil), versions...)}, Namespace: namespace, RuntimeNamespace: runtimeNamespace}
	for _, p := range pipelines {
		for k, v := range p.Tags {
			tag := model.PipelineTag{PipelineID: p.UUID, TagKey: k, TagValue: v}
			if err := budget.Add(tag); err != nil {
				return nil, err
			}
			b.PipelineTags = append(b.PipelineTags, tag)
		}
	}
	for _, v := range versions {
		for k, val := range v.Tags {
			tag := model.PipelineVersionTag{PipelineVersionID: v.UUID, TagKey: k, TagValue: val}
			if err := budget.Add(tag); err != nil {
				return nil, err
			}
			b.VersionTags = append(b.VersionTags, tag)
		}
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := readTransferRows(tx.Where(clause.Eq{Column: "Namespace", Value: namespace}).Limit(MaxTransferObjects+1), &b.Experiments, budget); err != nil {
			return err
		}
		var experimentIDs []string
		for _, e := range b.Experiments {
			experimentIDs = append(experimentIDs, e.UUID)
		}
		var runs []model.Run
		if len(experimentIDs) > 0 {
			values := make([]any, len(experimentIDs))
			for i, id := range experimentIDs {
				values[i] = id
			}
			if err := readTransferRows(tx.Where(clause.IN{Column: clause.Column{Name: "ExperimentUUID"}, Values: values}).Limit(MaxTransferObjects+1), &b.Schedules, budget); err != nil {
				return err
			}
			q := tx.Where(clause.IN{Column: clause.Column{Name: "ExperimentUUID"}, Values: values}).Where(clause.Gt{Column: "FinishedAtInSec", Value: 0}).Where(clause.Gte{Column: "FinishedAtInSec", Value: opts.CompletedAfter})
			if opts.CompletedBefore > 0 {
				q = q.Where(clause.Lt{Column: "FinishedAtInSec", Value: opts.CompletedBefore})
			}
			if err := readTransferRows(q.Limit(MaxTransferRuns+1), &runs, budget); err != nil {
				return err
			}
			if len(runs) > MaxTransferRuns {
				return util.NewInvalidInputError("export exceeds %d completed runs; narrow the completion time window", MaxTransferRuns)
			}
		}
		rows := 0
		for _, run := range runs {
			if !terminal(run) {
				continue
			}
			// Re-exporting imported history remains history at the next destination.
			run.ImportedFrom = ""
			run.ImportDigest = ""
			if err := budget.Reserve(128); err != nil {
				return err
			}
			e := Entry{Run: run}
			if err := findTransferRows(tx.Limit(MaxTransferRows+1), "RunUUID", []string{run.UUID}, &e.Tasks, budget); err != nil {
				return err
			}
			if err := findTransferRows(tx.Limit(MaxTransferRows+1), "RunUUID", []string{run.UUID}, &e.Links, budget); err != nil {
				return err
			}
			if err := findTransferRows(tx.Limit(MaxTransferRows+1), "RunUUID", []string{run.UUID}, &e.Metrics, budget); err != nil {
				return err
			}
			ids := []string{}
			for _, l := range e.Links {
				ids = append(ids, l.ArtifactID)
			}
			if err := findTransferRows(tx.Limit(MaxTransferRows+1), "UUID", unique(ids), &e.Artifacts, budget); err != nil {
				return err
			}
			rows += len(e.Tasks) + len(e.Links) + len(e.Metrics) + len(e.Artifacts)
			if rows > MaxTransferRows {
				return util.NewInvalidInputError("export exceeds %d history rows; narrow the completion time window", MaxTransferRows)
			}
			sortEntry(&e)
			b.Entries = append(b.Entries, e)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	for _, f := range normalize {
		if err := f(b); err != nil {
			return nil, err
		}
	}
	return b, ValidateNamespace(b, namespace, runtimeNamespace)
}

// ValidateNamespace checks the whole graph before any destination lookup/write.
func ValidateNamespace(b *NamespaceBundle, namespace, runtimeNamespace string) error {
	if b == nil || b.Format != TransferFormat || b.Namespace != namespace {
		return util.NewInvalidInputError("archive namespace or format does not match the destination")
	}
	if namespace != "" && (b.RuntimeNamespace != namespace || runtimeNamespace != namespace) {
		return util.NewInvalidInputError("archive runtime namespace does not match the destination")
	}
	if namespace == "" && b.RuntimeNamespace != runtimeNamespace {
		return util.NewInvalidInputError("single-user runtime namespace must match the destination")
	}
	if len(b.Experiments)+len(b.Pipelines)+len(b.Versions)+len(b.Schedules) > MaxTransferObjects {
		return util.NewInvalidInputError("archive exceeds %d catalog and schedule objects", MaxTransferObjects)
	}
	// Historical runs can reference deleted definitions only in the old admin
	// format. Self-service archives require resolvable, namespace-owned catalog.
	experiments := map[string]bool{}
	pipelines := map[string]bool{}
	versions := map[string]string{}
	for _, e := range b.Experiments {
		if e.Namespace != namespace {
			return util.NewInvalidInputError("experiment belongs to another namespace")
		}
		experiments[e.UUID] = true
	}
	for _, p := range b.Pipelines {
		if p.Namespace != namespace {
			return util.NewInvalidInputError("pipeline belongs to another namespace")
		}
		pipelines[p.UUID] = true
	}
	for _, v := range b.Versions {
		if !pipelines[v.PipelineId] || v.PipelineSpec == "" || v.PipelineSpecURI != "" {
			return util.NewInvalidInputError("pipeline version must have a namespace-owned parent and an inline definition")
		}
		versions[v.UUID] = v.PipelineId
	}
	for _, p := range b.Pipelines {
		if p.DefaultVersionId != "" && versions[p.DefaultVersionId] != p.UUID {
			return util.NewInvalidInputError("Default pipeline version must resolve to a version of its archived pipeline")
		}
	}
	ownedSpec := func(p model.PipelineSpec) bool {
		return (p.PipelineId == "" || pipelines[p.PipelineId]) && (p.PipelineVersionId == "" || (versions[p.PipelineVersionId] != "" && (p.PipelineId == "" || versions[p.PipelineVersionId] == p.PipelineId)))
	}
	rows := 0
	for _, e := range b.Entries {
		if e.Run.Namespace != runtimeNamespace || !ownedSpec(e.Run.PipelineSpec) {
			return util.NewInvalidInputError("run or referenced pipeline belongs to another namespace or is unresolved")
		}
		rows += len(e.Tasks) + len(e.Artifacts) + len(e.Links) + len(e.Metrics)
	}
	if rows > MaxTransferRows {
		return util.NewInvalidInputError("archive exceeds %d history rows", MaxTransferRows)
	}
	seen := map[string]bool{}
	for _, j := range b.Schedules {
		if j.UUID == "" || seen[j.UUID] || j.Namespace != runtimeNamespace || !experiments[j.ExperimentId] || !ownedSpec(j.PipelineSpec) {
			return util.NewInvalidInputError("schedule has duplicate identity or unresolved namespace ownership")
		}
		seen[j.UUID] = true
	}
	// The original validator expects logical and runtime namespace equality.
	// A shallow copy of experiments accommodates the established single-user
	// convention without weakening any task/artifact ownership checks.
	copyBundle := b.Bundle
	copyBundle.Experiments = append([]model.Experiment(nil), b.Experiments...)
	for i := range copyBundle.Experiments {
		copyBundle.Experiments[i].Namespace = runtimeNamespace
	}
	if err := validate(&copyBundle); err != nil {
		return util.NewInvalidInputError("Invalid namespace archive: %v", err)
	}
	return nil
}

// TransferPlan contains only mapped, validated resources. Stagers replace
// predicted IDs with the Kubernetes UIDs before the final SQL commit.
type TransferPlan struct {
	Bundle          *NamespaceBundle
	Receipts        []model.TransferReceipt
	Existing        map[string]bool
	ExternalCatalog bool
	Summary         transfer.Summary
}

func PrepareTransfer(ctx context.Context, db *gorm.DB, b *NamespaceBundle, opts transfer.ImportOptions) (*TransferPlan, error) {
	var destination string
	var err error
	if opts.DryRun {
		var identity model.TransferIdentity
		err = db.WithContext(ctx).Where(clause.Eq{Column: "Key", Value: "installation"}).Take(&identity).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			destination = uuid.NewString()
			err = nil
		} else {
			destination = identity.UUID
		}
	} else {
		destination, err = InstallationID(ctx, db)
	}
	if err != nil {
		return nil, err
	}
	p := &TransferPlan{Bundle: b, Existing: map[string]bool{}, Summary: transfer.Summary{DryRun: opts.DryRun, Counts: transfer.Counts{Experiments: len(b.Experiments), Pipelines: len(b.Pipelines), PipelineVersions: len(b.Versions), Runs: len(b.Entries), Schedules: len(b.Schedules)}}}
	if len(b.Pipelines) > 0 {
		p.Summary.Warnings = append(p.Summary.Warnings, "Existing explicit Kubernetes pipeline defaults are retained. SQL catalogs continue to select the newest version, including newly imported versions.")
	}
	if len(b.Schedules) > 0 {
		p.Summary.Warnings = append(p.Summary.Warnings, "Schedules are imported disabled with catchup disabled; enable them explicitly after reviewing service accounts and runtime settings.")
	}
	mappings := map[string]map[string]string{}
	add := func(kind, id string, value any) (string, error) {
		key, _ := digest([]string{b.Source, b.Namespace, kind, id})
		// Activity can advance while completion-time batches are exported.
		if experiment, ok := value.(model.Experiment); ok {
			experiment.LastRunCreatedAtInSec = 0
			value = experiment
		}
		if pipeline, ok := value.(model.Pipeline); ok {
			pipeline.DefaultVersionId = ""
			value = pipeline
		}
		if job, ok := value.(model.Job); ok {
			job.UpdatedAtInSec = 0
			job.Conditions = ""
			job.Enabled = false
			job.NoCatchup = true
			value = job
		}
		checksum, err := digest([]any{opts.NamePrefix, value})
		if err != nil {
			return "", err
		}
		target := uuid.NewSHA1(uuid.NameSpaceOID, []byte(destination+":"+key)).String()
		row := model.TransferReceipt{Key: key, Source: b.Source, Namespace: b.Namespace, Kind: kind, SourceID: id, TargetID: target, Digest: checksum}
		var existing model.TransferReceipt
		err = db.WithContext(ctx).Where(clause.Eq{Column: "Key", Value: key}).Take(&existing).Error
		switch {
		case err == nil:
			if existing.Digest != checksum {
				return "", util.NewInvalidInputError("previously imported %s definition changed; use its original archive or a separate destination", kind)
			}
			row.TargetID = existing.TargetID
			p.Existing[key] = true
			p.Summary.Skipped++
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return "", err
		default:
			p.Summary.Imported++
		}
		p.Receipts = append(p.Receipts, row)
		if mappings[kind] == nil {
			mappings[kind] = map[string]string{}
		}
		mappings[kind][id] = row.TargetID
		return row.TargetID, nil
	}
	for i := range b.Experiments {
		e := &b.Experiments[i]
		id, err := add("experiment", e.UUID, *e)
		if err != nil {
			return nil, err
		}
		e.UUID = id
		e.Name = opts.NamePrefix + e.Name
		if utf8.RuneCountInString(e.Name) > 128 {
			return nil, util.NewInvalidInputError("prefixed experiment name exceeds 128 characters")
		}
	}
	for i := range b.Pipelines {
		v := &b.Pipelines[i]
		id, err := add("pipeline", v.UUID, *v)
		if err != nil {
			return nil, err
		}
		v.UUID = id
		v.Status = model.PipelineReady
		v.Name = opts.NamePrefix + v.Name
		if utf8.RuneCountInString(v.Name) > 128 {
			return nil, util.NewInvalidInputError("prefixed pipeline name exceeds 128 characters")
		}
	}
	for i := range b.Versions {
		v := &b.Versions[i]
		id, err := add("version", v.UUID, *v)
		if err != nil {
			return nil, err
		}
		v.UUID = id
		v.Status = model.PipelineVersionReady
	}
	for i := range b.Schedules {
		j := &b.Schedules[i]
		id, err := add("schedule", j.UUID, *j)
		if err != nil {
			return nil, err
		}
		j.UUID = id
		j.K8SName = "transfer-" + id
		j.DisplayName = opts.NamePrefix + j.DisplayName
		j.Enabled = false
		j.NoCatchup = true
		j.Conditions = string(model.StatusStateDisabled)
	}
	for i := range b.Entries {
		e := &b.Entries[i]
		sortEntry(e)
		id, err := add("run", e.Run.UUID, *e)
		if err != nil {
			return nil, err
		}
		e.Run.UUID = id
	}
	mapped := func(kind, id string) string {
		if id == "" {
			return ""
		}
		if v := mappings[kind][id]; v != "" {
			return v
		}
		key, _ := digest([]string{b.Source, b.Namespace, kind, id})
		return uuid.NewSHA1(uuid.NameSpaceOID, []byte(destination+":"+key)).String()
	}
	spec := func(s *model.PipelineSpec) {
		s.PipelineId = mapped("pipeline", s.PipelineId)
		s.PipelineVersionId = mapped("version", s.PipelineVersionId)
	}
	for i := range b.Pipelines {
		b.Pipelines[i].DefaultVersionId = mapped("version", b.Pipelines[i].DefaultVersionId)
	}
	for i := range b.Versions {
		b.Versions[i].PipelineId = mapped("pipeline", b.Versions[i].PipelineId)
		b.Versions[i].Pipeline = model.Pipeline{}
	}
	for i := range b.PipelineTags {
		b.PipelineTags[i].PipelineID = mapped("pipeline", b.PipelineTags[i].PipelineID)
	}
	for i := range b.VersionTags {
		b.VersionTags[i].PipelineVersionID = mapped("version", b.VersionTags[i].PipelineVersionID)
	}
	for i := range b.Schedules {
		j := &b.Schedules[i]
		j.ExperimentId = mapped("experiment", j.ExperimentId)
		spec(&j.PipelineSpec)
	}
	for i := range b.Entries {
		e := &b.Entries[i]
		e.Run.ExperimentId = mapped("experiment", e.Run.ExperimentId)
		spec(&e.Run.PipelineSpec)
		e.Run.RecurringRunId = ""
		for j := range e.Tasks {
			t := &e.Tasks[j]
			t.UUID = mapped("task", t.UUID)
			t.RunUUID = e.Run.UUID
			// Imported history must not register a live execution lookup key.
			t.LogicalKey = nil
			if t.ParentTaskUUID != nil {
				parent := mapped("task", *t.ParentTaskUUID)
				t.ParentTaskUUID = &parent
			}
		}
		for j := range e.Artifacts {
			a := &e.Artifacts[j]
			a.UUID = mapped("artifact", a.UUID)
			a.IdentityKey = nil
		}
		for j := range e.Links {
			l := &e.Links[j]
			l.UUID = mapped("link", l.UUID)
			l.TaskID = mapped("task", l.TaskID)
			l.ArtifactID = mapped("artifact", l.ArtifactID)
			l.RunUUID = e.Run.UUID
		}
		for j := range e.Metrics {
			e.Metrics[j].RunUUID = e.Run.UUID
		}
	}
	return p, nil
}

func (p *TransferPlan) Receipt(kind, target string) *model.TransferReceipt {
	for i := range p.Receipts {
		r := &p.Receipts[i]
		if r.Kind == kind && r.TargetID == target {
			return r
		}
	}
	return nil
}
func (p *TransferPlan) ReplaceID(kind, old, id string) {
	if old == id {
		return
	}
	if r := p.Receipt(kind, old); r != nil {
		r.TargetID = id
	}
	b := p.Bundle
	switch kind {
	case "pipeline":
		for i := range b.Pipelines {
			if b.Pipelines[i].UUID == old {
				b.Pipelines[i].UUID = id
			}
		}
		for i := range b.Versions {
			if b.Versions[i].PipelineId == old {
				b.Versions[i].PipelineId = id
			}
		}
		for i := range b.PipelineTags {
			if b.PipelineTags[i].PipelineID == old {
				b.PipelineTags[i].PipelineID = id
			}
		}
	case "version":
		for i := range b.Versions {
			if b.Versions[i].UUID == old {
				b.Versions[i].UUID = id
			}
		}
		for i := range b.Pipelines {
			if b.Pipelines[i].DefaultVersionId == old {
				b.Pipelines[i].DefaultVersionId = id
			}
		}
		for i := range b.VersionTags {
			if b.VersionTags[i].PipelineVersionID == old {
				b.VersionTags[i].PipelineVersionID = id
			}
		}
	case "schedule":
		for i := range b.Schedules {
			if b.Schedules[i].UUID == old {
				b.Schedules[i].UUID = id
			}
		}
	}
	update := func(s *model.PipelineSpec) {
		if kind == "pipeline" && s.PipelineId == old {
			s.PipelineId = id
		}
		if kind == "version" && s.PipelineVersionId == old {
			s.PipelineVersionId = id
		}
	}
	for i := range b.Entries {
		update(&b.Entries[i].Run.PipelineSpec)
	}
	for i := range b.Schedules {
		update(&b.Schedules[i].PipelineSpec)
	}
}

// CommitTransfer merges SQL resources and receipts together. External catalog
// and schedule objects are staged disabled beforehand and safely reusable.
func CommitTransfer(ctx context.Context, db *gorm.DB, p *TransferPlan, dryRun bool) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unique receipts reject conflicting concurrent imports of a source resource.
		for _, r := range p.Receipts {
			if err := insertOrCheck(tx, &r); err != nil {
				return err
			}
		}
		b := p.Bundle.Bundle
		// Logical empty namespace is retained in SQL. The admin importer validates
		// through a temporary normalized copy, while actual rows keep their namespace.
		originalExperiments := b.Experiments
		if p.Bundle.Namespace == "" {
			b.Experiments = append([]model.Experiment(nil), b.Experiments...)
			for i := range b.Experiments {
				b.Experiments[i].Namespace = p.Bundle.RuntimeNamespace
			}
		}
		if err := validate(&b); err != nil {
			return err
		}
		b.Experiments = originalExperiments
		// Import's own validator has the legacy namespace convention. Insert the
		// namespace-owned experiments here and use an internal validated variant.
		if _, err := importValidated(ctx, tx, &b, ImportOptions{ExternalCatalog: p.ExternalCatalog}); err != nil {
			return err
		}
		for _, j := range p.Bundle.Schedules {
			receipt := p.Receipt("schedule", j.UUID)
			if receipt != nil && p.Existing[receipt.Key] {
				continue
			}
			if err := createRecord(tx, &j); err != nil {
				return err
			}
			if err := createRecord(tx, &model.RecurringRunState{JobUUID: j.UUID}); err != nil {
				return err
			}
		}
		if dryRun {
			return errDryRunRollback
		}
		return nil
	})
	if errors.Is(err, errDryRunRollback) {
		return nil
	}
	return err
}

func SortTransfer(b *NamespaceBundle) {
	sort.Slice(b.Experiments, func(i, j int) bool { return b.Experiments[i].UUID < b.Experiments[j].UUID })
	sort.Slice(b.Pipelines, func(i, j int) bool { return b.Pipelines[i].UUID < b.Pipelines[j].UUID })
	sort.Slice(b.Versions, func(i, j int) bool { return b.Versions[i].UUID < b.Versions[j].UUID })
	sort.Slice(b.Schedules, func(i, j int) bool { return b.Schedules[i].UUID < b.Schedules[j].UUID })
	sort.Slice(b.Entries, func(i, j int) bool { return b.Entries[i].Run.UUID < b.Entries[j].Run.UUID })
}
