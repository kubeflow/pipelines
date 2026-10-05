// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	kubernetesmodel "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"google.golang.org/protobuf/proto"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/history"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const transferReceiptTag = "kfp-transfer-receipt"
const transferDigestTag = "kfp-transfer-digest"
const transferReceiptAnnotation = "pipelines.kubeflow.org/transfer-receipt"
const transferDigestAnnotation = "pipelines.kubeflow.org/transfer-digest"

func (r *ResourceManager) transferDB() (*gorm.DB, error) {
	provider, ok := r.dBStatusStore.(interface{ TransferDB() (*gorm.DB, error) })
	if !ok {
		return nil, util.NewInternalServerError(errors.New("transfer database unavailable"), "Transfer is unavailable")
	}
	db, err := provider.TransferDB()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Transfer database is unavailable")
	}
	return db, nil
}
func transferRuntimeNamespace(namespace string) string {
	if namespace != "" {
		return namespace
	}
	return common.GetPodNamespace()
}
func (r *ResourceManager) ExportTransfer(ctx context.Context, namespace string, opts transfer.ExportOptions) ([]byte, error) {
	if namespace == "" && common.IsMultiUserMode() {
		return nil, util.NewInvalidInputError("A namespace is required")
	}
	db, err := r.transferDB()
	if err != nil {
		return nil, err
	}
	pipelines, versions, err := r.exportTransferCatalog(namespace)
	if err != nil {
		return nil, err
	}
	b, err := history.ExportNamespace(ctx, db, namespace, transferRuntimeNamespace(namespace), opts, pipelines, versions, r.normalizeTransferReferences)
	if err != nil {
		return nil, transferOperationError(err, "Cannot export the namespace")
	}
	history.SortTransfer(b)
	data, err := json.Marshal(b)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Cannot encode the transfer archive")
	}
	if len(data) > transfer.MaxArchiveBytes {
		return nil, util.NewInvalidInputError("Archive exceeds 256 MiB; narrow the completion time window")
	}
	return data, nil
}
func (r *ResourceManager) exportTransferCatalog(namespace string) ([]model.Pipeline, []model.PipelineVersion, error) {
	p, v, err := r.exportTransferCatalogScope(namespace, namespace)
	if err != nil {
		return nil, nil, err
	}
	if _, kubernetes := r.pipelineStore.(*storage.PipelineStoreKubernetes); namespace == "" && !kubernetes {
		shared, sharedVersions, err := r.exportTransferCatalogScope(namespace, model.NoNamespace)
		if err != nil {
			return nil, nil, err
		}
		p = append(p, shared...)
		v = append(v, sharedVersions...)
	}
	if len(p)+len(v) > history.MaxTransferObjects {
		return nil, nil, transferPrecondition("Catalog exceeds the transfer object limit")
	}
	return p, v, nil
}
func (r *ResourceManager) exportTransferCatalogScope(namespace, scope string) ([]model.Pipeline, []model.PipelineVersion, error) {
	_, kubernetes := r.pipelineStore.(*storage.PipelineStoreKubernetes)
	if kubernetes && namespace == "" {
		scope = common.GetPodNamespace()
	}
	opts, err := list.NewOptions(&model.Pipeline{}, 100, "", nil)
	if err != nil {
		return nil, nil, err
	}
	filter := &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: scope}}
	var pipelines []model.Pipeline
	var versions []model.PipelineVersion
	for {
		rows, _, next, err := r.pipelineStore.ListPipelines(filter, opts)
		if err != nil {
			return nil, nil, err
		}
		for _, row := range rows {
			p := *row
			p.Status = model.PipelineReady
			if p.Namespace == model.NoNamespace && namespace == "" {
				p.Namespace = ""
			}
			if kubernetes && namespace == "" && p.Namespace == scope {
				p.Namespace = ""
			}
			if p.Namespace != namespace {
				return nil, nil, transferPrecondition("Catalog contains a pipeline outside the selected namespace")
			}
			defaultVersion, defaultErr := r.pipelineStore.GetDefaultPipelineVersion(p.UUID)
			if defaultErr == nil {
				p.DefaultVersionId = defaultVersion.UUID
			} else if !util.IsUserErrorCodeMatch(defaultErr, codes.NotFound) {
				return nil, nil, defaultErr
			}
			p.Tags, err = r.pipelineStore.GetPipelineTags(p.UUID)
			if err != nil {
				return nil, nil, err
			}
			delete(p.Tags, transferReceiptTag)
			delete(p.Tags, transferDigestTag)
			pipelines = append(pipelines, p)
			vo, err := list.NewOptions(&model.PipelineVersion{}, 100, "", nil)
			if err != nil {
				return nil, nil, err
			}
			for {
				vrows, _, vn, err := r.pipelineStore.ListPipelineVersions(p.UUID, vo)
				if err != nil {
					return nil, nil, err
				}
				for _, row := range vrows {
					v := *row
					v.Status = model.PipelineVersionReady
					data, _, err := r.fetchTemplateFromPipelineVersion(&v)
					if err != nil {
						return nil, nil, err
					}
					v.PipelineSpec = model.LargeText(data)
					v.PipelineSpecURI = ""
					v.Pipeline = model.Pipeline{}
					v.Tags, err = r.pipelineStore.GetPipelineVersionTags(v.UUID)
					if err != nil {
						return nil, nil, err
					}
					delete(v.Tags, transferReceiptTag)
					delete(v.Tags, transferDigestTag)
					versions = append(versions, v)
				}
				if len(pipelines)+len(versions) > history.MaxTransferObjects {
					return nil, nil, transferPrecondition("Catalog exceeds the transfer limit of %d objects", history.MaxTransferObjects)
				}
				if vn == "" {
					break
				}
				vo, err = list.NewOptionsFromToken(vn, 100)
				if err != nil {
					return nil, nil, err
				}
			}
		}
		if next == "" {
			break
		}
		opts, err = list.NewOptionsFromToken(next, 100)
		if err != nil {
			return nil, nil, err
		}
	}
	return pipelines, versions, nil
}

func (r *ResourceManager) ImportTransfer(ctx context.Context, namespace string, archive []byte, opts transfer.ImportOptions) (transfer.Summary, error) {
	empty := transfer.Summary{}
	if namespace == "" && common.IsMultiUserMode() {
		return empty, util.NewInvalidInputError("A namespace is required")
	}
	if len(archive) > transfer.MaxArchiveBytes {
		return empty, util.NewInvalidInputError("Archive exceeds 256 MiB")
	}
	var b history.NamespaceBundle
	decoder := json.NewDecoder(bytes.NewReader(archive))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&b); err != nil {
		return empty, util.NewInvalidInputErrorWithDetails(err, "Invalid transfer archive")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return empty, util.NewInvalidInputError("Archive must contain exactly one JSON object")
	}
	if err := history.ValidateNamespace(&b, namespace, transferRuntimeNamespace(namespace)); err != nil {
		return empty, util.NewInvalidInputError("Invalid transfer archive: %v", err)
	}
	db, err := r.transferDB()
	if err != nil {
		return empty, err
	}
	plan, err := history.PrepareTransfer(ctx, db, &b, opts)
	if err != nil {
		return empty, transferOperationError(err, "Cannot prepare the transfer")
	}
	_, plan.ExternalCatalog = r.pipelineStore.(*storage.PipelineStoreKubernetes)
	// Preview validates names/templates/permissions without external writes. The
	// SQL merge runs inside a rollback transaction to catch relational conflicts.
	if err := r.stageTransferCatalog(plan, false); err != nil {
		return empty, err
	}
	if err := r.stageTransferSchedules(ctx, plan, false); err != nil {
		return empty, err
	}
	if err := history.CommitTransfer(ctx, db, plan, true); err != nil {
		return empty, transferOperationError(err, "Cannot validate the transfer against destination data")
	}
	if opts.DryRun {
		return plan.Summary, nil
	}
	if err := r.stageTransferCatalog(plan, true); err != nil {
		return empty, err
	}
	if err := r.stageTransferSchedules(ctx, plan, true); err != nil {
		return empty, err
	}
	if err := history.CommitTransfer(ctx, db, plan, false); err != nil {
		return empty, util.NewInternalServerError(err, "Transfer was not committed; retry the same archive to reuse safely staged resources")
	}
	return plan.Summary, nil
}

func tagsWithReceipt(tags map[string]string, r *model.TransferReceipt) map[string]string {
	copy := map[string]string{}
	for k, v := range tags {
		copy[k] = v
	}
	copy[transferReceiptTag] = r.Key[:63]
	copy[transferDigestTag] = r.Digest[:63]
	return copy
}
func matchesTransferTags(tags map[string]string, r *model.TransferReceipt) bool {
	return tags[transferReceiptTag] == r.Key[:63] && tags[transferDigestTag] == r.Digest[:63]
}

func (r *ResourceManager) stageTransferCatalog(plan *history.TransferPlan, apply bool) error {
	if !plan.ExternalCatalog {
		return nil
	}
	b := plan.Bundle
	for i := range b.Pipelines {
		p := b.Pipelines[i]
		receipt := plan.Receipt("pipeline", p.UUID)
		if errs := validation.IsDNS1123Subdomain(p.Name); len(errs) > 0 {
			return util.NewInvalidInputError("Pipeline name %q is not a Kubernetes resource name", p.Name)
		}
		existing, err := r.pipelineStore.GetPipelineByNameAndNamespace(p.Name, p.Namespace)
		if err == nil {
			tags, err := r.pipelineStore.GetPipelineTags(existing.UUID)
			if err != nil {
				return err
			}
			if !matchesTransferTags(tags, receipt) || existing.Name != p.Name || existing.Description != p.Description {
				return util.NewAlreadyExistError("Pipeline name %q conflicts with destination catalog", p.Name)
			}
			plan.ReplaceID("pipeline", p.UUID, existing.UUID)
			continue
		}
		if !util.IsUserErrorCodeMatch(err, codes.NotFound) {
			return err
		}
		if plan.Existing[receipt.Key] {
			return transferPrecondition("A previously imported pipeline has been deleted")
		}
		if !apply {
			continue
		}
		p.Tags = tagsWithReceipt(p.Tags, receipt)
		p.DefaultVersionId = ""
		created, err := r.pipelineStore.CreatePipeline(&p)
		if err != nil {
			return err
		}
		plan.ReplaceID("pipeline", receipt.TargetID, created.UUID)
	}
	for i := range b.Versions {
		v := b.Versions[i]
		receipt := plan.Receipt("version", v.UUID)
		if errs := validation.IsDNS1123Subdomain(v.Name); len(errs) > 0 {
			return util.NewInvalidInputError("Pipeline version name %q is not a Kubernetes resource name", v.Name)
		}
		var parentModel model.Pipeline
		for _, p := range b.Pipelines {
			if p.UUID == v.PipelineId {
				parentModel = p
				break
			}
		}
		if _, err := kubernetesmodel.FromPipelineVersionModel(parentModel, v); err != nil {
			return util.NewInvalidInputErrorWithDetails(err, "Pipeline version cannot be represented in the destination Kubernetes catalog")
		}
		// During preview a new parent has no CR yet; its version cannot collide.
		parent := plan.Receipt("pipeline", v.PipelineId)
		if !apply && parent != nil && !plan.Existing[parent.Key] {
			if _, err := r.pipelineStore.GetPipeline(v.PipelineId); util.IsUserErrorCodeMatch(err, codes.NotFound) {
				continue
			} else if err != nil {
				return err
			}
		}
		existing, err := r.pipelineStore.GetPipelineVersionByName(v.PipelineId, v.Name)
		if err == nil {
			tags, err := r.pipelineStore.GetPipelineVersionTags(existing.UUID)
			if err != nil {
				return err
			}
			if !matchesTransferTags(tags, receipt) || !sameTransferSpec(existing.PipelineSpec, v.PipelineSpec) || existing.PipelineId != v.PipelineId {
				return util.NewAlreadyExistError("Pipeline version name %q conflicts with destination catalog", v.Name)
			}
			plan.ReplaceID("version", v.UUID, existing.UUID)
			continue
		}
		if !util.IsUserErrorCodeMatch(err, codes.NotFound) {
			return err
		}
		if plan.Existing[receipt.Key] {
			return transferPrecondition("A previously imported pipeline version has been deleted")
		}
		if !apply {
			continue
		}
		v.Tags = tagsWithReceipt(v.Tags, receipt)
		v.UUID = "" // The destination API server assigns the Kubernetes UID.
		created, err := r.pipelineStore.CreatePipelineVersion(&v)
		if err != nil {
			return err
		}
		plan.ReplaceID("version", receipt.TargetID, created.UUID)
	}
	if apply {
		store := r.pipelineStore.(*storage.PipelineStoreKubernetes)
		for _, p := range b.Pipelines {
			receipt := plan.Receipt("pipeline", p.UUID)
			if p.DefaultVersionId != "" && !plan.Existing[receipt.Key] {
				if err := store.SetTransferDefaultVersion(p.UUID, p.DefaultVersionId); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func sameTransferSpec(a, b model.LargeText) bool {
	x, err := template.NewV2SpecTemplate([]byte(a), template.TemplateOptions{})
	if err != nil {
		return false
	}
	y, err := template.NewV2SpecTemplate([]byte(b), template.TemplateOptions{})
	if err != nil {
		return false
	}
	return proto.Equal(x.PipelineSpec(), y.PipelineSpec()) && proto.Equal(x.PlatformSpec(), y.PlatformSpec())
}

func (r *ResourceManager) transferScheduleWorkflow(ctx context.Context, plan *history.TransferPlan, job *model.Job) (*scheduledworkflow.ScheduledWorkflow, error) {
	validationJob := *job
	// Validate against the archive's resolved catalog before SQL catalog rows exist.
	var version *model.PipelineVersion
	targetVersion := job.PipelineVersionId
	if targetVersion == "" {
		for _, p := range plan.Bundle.Pipelines {
			if p.UUID == job.PipelineId {
				targetVersion = p.DefaultVersionId
				break
			}
		}
	}
	for i := range plan.Bundle.Versions {
		v := &plan.Bundle.Versions[i]
		if (targetVersion != "" && v.UUID == targetVersion) || (targetVersion == "" && v.PipelineId == job.PipelineId && (version == nil || v.CreatedAtInSec > version.CreatedAtInSec)) {
			version = v
		}
	}
	if version != nil {
		validationJob.PipelineSpecManifest = version.PipelineSpec
	}
	validationJob.PipelineId = ""
	validationJob.PipelineVersionId = ""
	prepared, err := r.prepareJobWorkflow(ctx, &validationJob)
	if err != nil {
		return nil, err
	}
	job.ServiceAccount = validationJob.ServiceAccount
	var swf *scheduledworkflow.ScheduledWorkflow
	if job.PipelineId == "" && job.PipelineVersionId == "" && !common.IsMultiUserMode() {
		// The single-user controller cannot send an inline source over the generic
		// run RPC. Retain the normal compiled schedule path for inline-only jobs.
		if prepared.Spec.Workflow == nil || prepared.Spec.Workflow.Spec == nil {
			return nil, transferPrecondition("Inline-only schedules with plugins require saving a catalog pipeline before transfer")
		}
		swf = prepared
	} else {
		swf, err = template.NewGenericScheduledWorkflow(job)
		if err != nil {
			return nil, err
		}
		parameters, err := template.StringMapToCRDParameters(string(job.RuntimeConfig.Parameters))
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Invalid schedule parameters")
		}
		swf.Spec.Workflow = &scheduledworkflow.WorkflowResource{Parameters: parameters, PipelineRoot: string(job.PipelineRoot)}
	}

	swf.Spec.Enabled = false
	swf.Spec.NoCatchup = util.BoolPointer(true)
	swf.Namespace = job.Namespace
	swf.Name = job.K8SName
	swf.GenerateName = ""
	return swf, nil
}
func (r *ResourceManager) stageTransferSchedules(ctx context.Context, plan *history.TransferPlan, apply bool) error {
	for i := range plan.Bundle.Schedules {
		job := &plan.Bundle.Schedules[i]
		receipt := plan.Receipt("schedule", job.UUID)
		job.K8SName = "transfer-" + receipt.Key[:40]
		desired, err := r.transferScheduleWorkflow(ctx, plan, job)
		if err != nil {
			return err
		}
		desired.Annotations = map[string]string{transferReceiptAnnotation: receipt.Key, transferDigestAnnotation: receipt.Digest}
		client := r.getScheduledWorkflowClient(job.Namespace)
		existing, err := client.Get(ctx, job.K8SName, metav1.GetOptions{})
		if err == nil {
			if existing.Annotations[transferReceiptAnnotation] != receipt.Key || existing.Annotations[transferDigestAnnotation] != receipt.Digest {
				return util.NewAlreadyExistError("An unrelated schedule occupies the imported schedule name")
			}
			if plan.Existing[receipt.Key] {
				if string(existing.UID) != job.UUID {
					return transferPrecondition("Imported schedule identity changed")
				}
				continue
			}
			if existing.Spec.Enabled || !reflect.DeepEqual(existing.Spec, desired.Spec) {
				return transferPrecondition("A staged schedule was changed before its import completed")
			}
			plan.ReplaceID("schedule", job.UUID, string(existing.UID))
			continue
		}
		if !apierrors.IsNotFound(err) {
			return util.NewInternalServerError(err, "Cannot inspect destination schedules")
		}
		if plan.Existing[receipt.Key] {
			return transferPrecondition("A previously imported schedule has been deleted")
		}
		if !apply {
			continue
		}
		created, err := client.Create(ctx, desired)
		if apierrors.IsAlreadyExists(err) {
			return util.NewFailedPreconditionError(err, "Another transfer staged this schedule; retry the archive")
		}
		if err != nil {
			return util.NewInternalServerError(err, "Cannot stage a disabled schedule; retry the same archive")
		}
		if created.UID == "" {
			return util.NewInternalServerError(fmt.Errorf("missing ScheduledWorkflow UID"), "Cannot stage a disabled schedule")
		}
		plan.ReplaceID("schedule", job.UUID, string(created.UID))
	}
	return nil
}

func transferOperationError(err error, message string) error {
	var userError *util.UserError
	if errors.As(err, &userError) {
		return err
	}
	return util.NewInternalServerError(err, "%s", message)
}

// Deleted catalog entries are safe to omit when a run retained its inline spec.
// Existing resources outside the selected namespace are never copied or treated
// as deleted; the whole export is rejected without disclosing their contents.
func (r *ResourceManager) normalizeTransferReferences(b *history.NamespaceBundle) error {
	pipelines := map[string]bool{}
	versions := map[string]bool{}
	for _, p := range b.Pipelines {
		pipelines[p.UUID] = true
	}
	for _, v := range b.Versions {
		versions[v.UUID] = true
	}
	normalize := func(s *model.PipelineSpec) error {
		missing := false
		if s.PipelineId != "" && !pipelines[s.PipelineId] {
			_, err := r.pipelineStore.GetPipeline(s.PipelineId)
			if !util.IsUserErrorCodeMatch(err, codes.NotFound) {
				if err != nil {
					return err
				}
				return transferPrecondition("A referenced pipeline is outside the selected catalog; check namespace ownership and retry")
			}
			missing = true
			s.PipelineId = ""
		}
		if s.PipelineVersionId != "" && !versions[s.PipelineVersionId] {
			_, err := r.pipelineStore.GetPipelineVersion(s.PipelineVersionId)
			if !util.IsUserErrorCodeMatch(err, codes.NotFound) {
				if err != nil {
					return err
				}
				return transferPrecondition("A referenced pipeline version is outside the selected catalog; check namespace ownership and retry")
			}
			missing = true
			s.PipelineVersionId = ""
		}
		if missing && s.PipelineSpecManifest == "" {
			return transferPrecondition("A deleted pipeline definition has no retained inline specification")
		}
		return nil
	}
	for i := range b.Entries {
		if err := normalize(&b.Entries[i].Run.PipelineSpec); err != nil {
			return err
		}
	}
	for i := range b.Schedules {
		if err := normalize(&b.Schedules[i].PipelineSpec); err != nil {
			return err
		}
	}
	return nil
}

func transferPrecondition(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	return util.NewFailedPreconditionError(errors.New(message), "%s", message)
}
