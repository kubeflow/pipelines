// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

// Package resource manages persisted pipeline resources and their Kubernetes lifecycle.
package resource

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type transferDBProvider interface{ TransferDB() (*gorm.DB, error) }

func (r *ResourceManager) transferEngine(ctx context.Context) (*transfer.Engine, func(), error) {
	if r.transferDB == nil {
		return nil, func() {}, util.NewFailedPreconditionError(errors.New("database access unavailable"), "Transfer is unavailable in this API server")
	}
	db, err := r.transferDB()
	if err != nil {
		return nil, func() {}, err
	}
	engine := &transfer.Engine{DB: db, Schedules: transferSchedules{r: r}, RuntimeNamespace: common.GetPodNamespace(), LoadPipelineSpec: func(_ context.Context, v *model.PipelineVersion) ([]byte, error) {
		data, _, err := r.fetchTemplateFromPipelineVersion(v)
		return data, err
	}}
	engine.ValidatePipelineSpec = func(spec []byte) error {
		if _, err := template.New(spec, template.TemplateOptions{}); err != nil {
			return util.NewInvalidInputError("Invalid pipeline definition in transfer archive: %v", err)
		}
		return nil
	}
	if store, ok := r.pipelineStore.(*storage.PipelineStoreKubernetes); ok {
		engine.Catalog = storage.TransferCatalog{Store: store, RuntimeNamespace: common.GetPodNamespace()}
	}
	var tlsConfig *tls.Config
	if common.GetMetadataTLSEnabled() {
		tlsConfig, err = util.GetTLSConfig(common.CustomCaCertPath)
		if err != nil {
			return nil, func() {}, err
		}
	}
	creds := insecure.NewCredentials()
	if tlsConfig != nil {
		creds = credentials.NewTLS(tlsConfig)
	}
	address := common.GetMetadataServiceName() + ":8080"
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(creds), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(transfer.MaxArchiveBytes), grpc.MaxCallSendMsgSize(transfer.MaxArchiveBytes)))
	if err != nil {
		return nil, func() {}, err
	}
	engine.Metadata = &transfer.Metadata{RPC: transfer.ProtoRPC{Conn: conn}}
	return engine, func() { conn.Close() }, nil
}

// ExportTransfer exports namespace-owned definitions and completed history.
func (r *ResourceManager) ExportTransfer(ctx context.Context, namespace string, opts transfer.ExportOptions) ([]byte, error) {
	e, close, err := r.transferEngine(ctx)
	if err != nil {
		return nil, err
	}
	defer close()
	return e.Export(ctx, namespace, opts)
}

// ImportTransfer validates and publishes an archive without running a pipeline.
func (r *ResourceManager) ImportTransfer(ctx context.Context, namespace string, archive []byte, opts transfer.ImportOptions) (transfer.Summary, error) {
	e, close, err := r.transferEngine(ctx)
	if err != nil {
		return transfer.Summary{}, err
	}
	defer close()
	return e.Import(ctx, namespace, archive, opts)
}

type transferSchedules struct{ r *ResourceManager }

func (a transferSchedules) Prepare(ctx context.Context, source string, original *model.Job, digest string, validationSpec []byte, dry bool) (*model.Job, error) {
	j := *original
	j.Enabled = false
	j.NoCatchup = true
	keyBytes, _ := json.Marshal([]string{source, j.Namespace, j.UUID})
	sum := sha256.Sum256(keyBytes)
	name := "transfer-" + hex.EncodeToString(sum[:])[:40]
	validation := j
	validation.PipelineId = ""
	validation.PipelineVersionId = ""
	validation.PipelineName = ""
	validation.PipelineSpecManifest = model.LargeText(validationSpec)
	validation.WorkflowSpecManifest = ""
	swf, tmpl, _, err := a.r.prepareJobWorkflow(ctx, &validation)
	if err != nil {
		return nil, err
	}
	if v2, ok := tmpl.(*template.V2Spec); ok {
		if err := v2.ValidateJobInputs(&validation); err != nil {
			return nil, err
		}
	}
	// Restore destination catalog references for API-based schedules. Compiled
	// schedules retain the validated workflow and plugin behavior of CreateJob.
	swf.Spec.PipelineId = j.PipelineId
	swf.Spec.PipelineVersionId = j.PipelineVersionId
	swf.Spec.PipelineName = j.PipelineName
	serviceAccount := swf.Spec.ServiceAccount
	if serviceAccount == "" && tmpl.GetTemplateType() == template.V1 {
		compiled, err := tmpl.ScheduledWorkflow(&validation)
		if err != nil {
			return nil, err
		}
		if compiled.Spec.Workflow != nil {
			if execution, err := util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, compiled.Spec.Workflow); err == nil {
				serviceAccount = execution.ServiceAccount()
			}
		}
	}
	if serviceAccount == "" && swf.Spec.Workflow != nil {
		if execution, err := util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, swf.Spec.Workflow); err == nil {
			serviceAccount = execution.ServiceAccount()
		}
	}
	if serviceAccount == "" {
		serviceAccount = j.ServiceAccount
	}
	if err := a.r.authorizeServiceAccount(ctx, serviceAccount, j.Namespace); err != nil {
		return nil, err
	}
	j.ServiceAccount = serviceAccount
	if j.PipelineId != "" || j.PipelineVersionId != "" {
		swf, err = template.NewGenericScheduledWorkflow(&j)
		if err != nil {
			return nil, err
		}
		parameters, err := template.StringMapToCRDParameters(string(j.RuntimeConfig.Parameters))
		if err != nil {
			return nil, err
		}
		swf.Spec.Workflow = &scheduledworkflow.WorkflowResource{Parameters: parameters, PipelineRoot: string(j.PipelineRoot)}
		swf.Spec.ServiceAccount = serviceAccount
	} else if !common.IsMultiUserMode() && a.r.pluginDispatcher.PluginsRegistered() {
		return nil, util.NewFailedPreconditionError(errors.New("inline-only schedules require a catalog definition with plugins"), "Upload the inline pipeline definition to the catalog before transferring this schedule")
	}
	swf.Name = name
	swf.GenerateName = ""
	swf.Namespace = j.Namespace
	swf.Spec.Enabled = false
	swf.Spec.NoCatchup = util.BoolPointer(true)
	swf.Annotations = map[string]string{"pipelines.kubeflow.org/transfer-source": source, "pipelines.kubeflow.org/transfer-id": original.UUID, "pipelines.kubeflow.org/transfer-digest": digest}
	client := a.r.getScheduledWorkflowClient(j.Namespace)
	current, err := client.Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		for k, v := range swf.Annotations {
			if current.Annotations[k] != v {
				return nil, util.NewAlreadyExistError("Schedule staging name already has different provenance")
			}
		}
		if current.Spec.Enabled || !reflect.DeepEqual(current.Spec, swf.Spec) {
			return nil, util.NewAlreadyExistError("Staged schedule changed; restore it before retrying the transfer")
		}
		j.UUID = string(current.UID)
		j.K8SName = current.Name
		return &j, nil
	}
	if !apierrors.IsNotFound(err) && !util.IsNotFound(err) {
		return nil, err
	}
	if dry {
		return &j, nil
	}
	created, err := client.Create(ctx, swf)
	if err != nil {
		return nil, err
	}
	if created.UID == "" {
		return nil, fmt.Errorf("kubernetes did not assign a schedule ID")
	}
	j.UUID = string(created.UID)
	j.K8SName = created.Name
	j.Conditions = model.StatusStateDisabled.ToString()
	return &j, nil
}

// transferScheduleIDs prevents startup reconciliation from replacing imported
// API-based schedules with source compiled workflows. The local CR is authoritative.
func (r *ResourceManager) transferScheduleIDs(ctx context.Context) (map[string]bool, error) {
	result := map[string]bool{}
	if r.transferDB == nil {
		return result, nil
	}
	db, err := r.transferDB()
	if err != nil {
		return nil, err
	}
	var receipts []model.TransferReceipt
	if err := db.WithContext(ctx).Where(map[string]any{"Kind": "schedule"}).Find(&receipts).Error; err != nil {
		return nil, err
	}
	for _, receipt := range receipts {
		result[receipt.TargetID] = true
	}
	return result, nil
}
