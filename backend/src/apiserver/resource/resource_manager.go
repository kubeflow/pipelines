// Copyright 2018 The Kubeflow Authors
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

package resource

import (
	containerlist "container/list"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	kfpauth "github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	exec "github.com/kubeflow/pipelines/backend/src/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	scheduledworkflowclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/retry"
)

const (
	workflowReportRejectionIdentityMismatch    = "identity_mismatch"
	workflowReportRejectionNamespaceMismatch   = "namespace_mismatch"
	workflowReportRejectionOwnershipUnresolved = "ownership_unresolved"
	storedWorkflowIdentityCacheCapacity        = 10_000
)

// Metric variables. Please prefix the metric names with resource_manager_.
var (
	extraLabels = []string{
		// display in which Kubeflow namespace the runs were triggered
		"profile",

		// display workflow name
		"workflow",
	}

	// Count the removed workflows due to garbage collection.
	workflowGCCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "resource_manager_workflow_gc",
		Help: "The number of garbage-collected workflows",
	})

	// Count reports rejected before they can mutate a run or delete a Workflow.
	// The reason label is restricted to constants so the metric has
	// bounded cardinality and can be used for operator alerts.
	workflowReportRejectedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "resource_manager_workflow_reports_rejected_total",
		Help: "The number of workflow reports rejected before persistence or garbage collection",
	}, []string{"reason"})

	// Count the successful workflow runs
	workflowSuccessCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_manager_workflow_runs_success",
		Help: "The current number of successful workflow runs",
	}, extraLabels)

	// Count the failed workflow runs
	workflowFailedCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_manager_workflow_runs_failed",
		Help: "The current number of failed workflow runs",
	}, extraLabels)

	// Gap in seconds between creating an execution spec (Argo or other backend) for a recurring run and reporting it via the persistence agent.
	recurringPipelineRunReportGap = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "resource_manager_recurring_run_report_gap",
		Help:    "Recurring Run Report Delay",
		Buckets: prometheus.ExponentialBuckets(0.5, 2, 10), // 0.5s -> 4min
	})

	// Map API enum values to Kubernetes DeletionPropagation values
	propagationPolicyMap = map[apiv2beta1.DeletePropagationPolicy]v1.DeletionPropagation{
		apiv2beta1.DeletePropagationPolicy_FOREGROUND: v1.DeletePropagationForeground,
		apiv2beta1.DeletePropagationPolicy_BACKGROUND: v1.DeletePropagationBackground,
		apiv2beta1.DeletePropagationPolicy_ORPHAN:     v1.DeletePropagationOrphan,
	}
)

type ClientManagerInterface interface {
	ExperimentStore() storage.ExperimentStoreInterface
	PipelineStore() storage.PipelineStoreInterface
	JobStore() storage.JobStoreInterface
	RunStore() storage.RunStoreInterface
	TaskStore() storage.TaskStoreInterface
	ResourceReferenceStore() storage.ResourceReferenceStoreInterface
	DBStatusStore() storage.DBStatusStoreInterface
	DefaultExperimentStore() storage.DefaultExperimentStoreInterface
	ObjectStore() storage.ObjectStore
	ExecClient() util.ExecutionClient
	SwfClient() client.SwfClientInterface
	KubernetesCoreClient() client.KubernetesCoreInterface
	SubjectAccessReviewClient() client.SubjectAccessReviewInterface
	TokenReviewClient() client.TokenReviewInterface
	LogArchive() archive.LogArchiveInterface
	Time() util.TimeInterface
	UUID() util.UUIDGeneratorInterface
	Authenticators() []kfpauth.Authenticator
}

type ResourceManagerOptions struct {
	CollectMetrics       bool                              `json:"collect_metrics,omitempty"`
	CacheDisabled        bool                              `json:"cache_disabled,omitempty"`
	DefaultWorkspace     *corev1.PersistentVolumeClaimSpec `json:"default_workspace,omitempty"`
	MLPipelineTLSEnabled bool                              `json:"ml_pipeline_tls_enabled,omitempty"`
	DefaultRunAsUser     *int64                            `json:"default_run_as_user,omitempty"`
	DefaultRunAsGroup    *int64                            `json:"default_run_as_group,omitempty"`
	DefaultRunAsNonRoot  *bool                             `json:"default_run_as_non_root,omitempty"`
	DefaultHostUsers     *bool                             `json:"default_host_users,omitempty"`
}

type ResourceManager struct {
	experimentStore           storage.ExperimentStoreInterface
	pipelineStore             storage.PipelineStoreInterface
	jobStore                  storage.JobStoreInterface
	runStore                  storage.RunStoreInterface
	taskStore                 storage.TaskStoreInterface
	resourceReferenceStore    storage.ResourceReferenceStoreInterface
	dBStatusStore             storage.DBStatusStoreInterface
	defaultExperimentStore    storage.DefaultExperimentStoreInterface
	objectStore               storage.ObjectStore
	execClient                util.ExecutionClient
	swfClient                 client.SwfClientInterface
	k8sCoreClient             client.KubernetesCoreInterface
	subjectAccessReviewClient client.SubjectAccessReviewInterface
	tokenReviewClient         client.TokenReviewInterface
	logArchive                archive.LogArchiveInterface
	time                      util.TimeInterface
	uuid                      util.UUIDGeneratorInterface
	authenticators            []kfpauth.Authenticator
	options                   *ResourceManagerOptions
	pluginDispatcher          apiserverPlugins.RunPluginDispatcher
	storedWorkflowIdentities  storedWorkflowIdentityCache
}

type storedWorkflowIdentity struct {
	name            string
	namespace       string
	uid             types.UID
	retryGeneration int64
	manifestDigest  [sha256.Size]byte
}

type cachedStoredWorkflowIdentity struct {
	identity storedWorkflowIdentity
	element  *containerlist.Element
}

type storedWorkflowIdentityCache struct {
	mu      sync.Mutex
	entries map[string]cachedStoredWorkflowIdentity
	recency *containerlist.List
}

func (c *storedWorkflowIdentityCache) load(runID string) (storedWorkflowIdentity, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, found := c.entries[runID]
	if !found {
		return storedWorkflowIdentity{}, false
	}
	c.recency.MoveToBack(entry.element)
	return entry.identity, true
}

func (c *storedWorkflowIdentityCache) loadOrStore(
	runID string,
	identity storedWorkflowIdentity,
) storedWorkflowIdentity {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, found := c.entries[runID]; found {
		c.recency.MoveToBack(entry.element)
		// Retry generations are monotonic. A report that loaded an older run
		// row must never replace the identity cached by a completed RetryRun.
		if entry.identity.retryGeneration > identity.retryGeneration {
			return entry.identity
		}
		// Cache entries are valid only for the exact persisted manifest they
		// were decoded from. Identity repairs can update that manifest without
		// incrementing the retry generation, including from another replica.
		if entry.identity.retryGeneration == identity.retryGeneration &&
			entry.identity.manifestDigest == identity.manifestDigest {
			return entry.identity
		}
		entry.identity = identity
		c.entries[runID] = entry
		return identity
	}
	if c.entries == nil {
		c.entries = make(map[string]cachedStoredWorkflowIdentity)
		c.recency = containerlist.New()
	}
	if len(c.entries) >= storedWorkflowIdentityCacheCapacity {
		oldest := c.recency.Front()
		delete(c.entries, oldest.Value.(string))
		c.recency.Remove(oldest)
	}
	element := c.recency.PushBack(runID)
	c.entries[runID] = cachedStoredWorkflowIdentity{identity: identity, element: element}
	return identity
}

func (c *storedWorkflowIdentityCache) replaceAfterPersist(
	runID string,
	expectedManifestDigest [sha256.Size]byte,
	identity storedWorkflowIdentity,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, found := c.entries[runID]; found {
		if entry.identity.retryGeneration > identity.retryGeneration {
			c.recency.MoveToBack(entry.element)
			return
		}
		// A later report can commit and refresh the cache before an earlier
		// reporter resumes after its own commit. Only advance an entry from the
		// manifest this write replaced (or accept the exact new manifest), so
		// delayed post-commit work cannot restore an older cache digest.
		if entry.identity.retryGeneration == identity.retryGeneration &&
			entry.identity.manifestDigest != expectedManifestDigest &&
			entry.identity.manifestDigest != identity.manifestDigest {
			c.recency.MoveToBack(entry.element)
			return
		}
		entry.identity = identity
		c.entries[runID] = entry
		c.recency.MoveToBack(entry.element)
		return
	}
	if c.entries == nil {
		c.entries = make(map[string]cachedStoredWorkflowIdentity)
		c.recency = containerlist.New()
	}
	if len(c.entries) >= storedWorkflowIdentityCacheCapacity {
		oldest := c.recency.Front()
		delete(c.entries, oldest.Value.(string))
		c.recency.Remove(oldest)
	}
	element := c.recency.PushBack(runID)
	c.entries[runID] = cachedStoredWorkflowIdentity{identity: identity, element: element}
}

func (c *storedWorkflowIdentityCache) delete(runID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, found := c.entries[runID]
	if !found {
		return
	}
	delete(c.entries, runID)
	c.recency.Remove(entry.element)
}

func NewResourceManager(clientManager ClientManagerInterface, options *ResourceManagerOptions) *ResourceManager {
	rm := &ResourceManager{
		experimentStore:           clientManager.ExperimentStore(),
		pipelineStore:             clientManager.PipelineStore(),
		jobStore:                  clientManager.JobStore(),
		runStore:                  clientManager.RunStore(),
		taskStore:                 clientManager.TaskStore(),
		resourceReferenceStore:    clientManager.ResourceReferenceStore(),
		dBStatusStore:             clientManager.DBStatusStore(),
		defaultExperimentStore:    clientManager.DefaultExperimentStore(),
		objectStore:               clientManager.ObjectStore(),
		execClient:                clientManager.ExecClient(),
		swfClient:                 clientManager.SwfClient(),
		k8sCoreClient:             clientManager.KubernetesCoreClient(),
		subjectAccessReviewClient: clientManager.SubjectAccessReviewClient(),
		tokenReviewClient:         clientManager.TokenReviewClient(),
		logArchive:                clientManager.LogArchive(),
		time:                      clientManager.Time(),
		uuid:                      clientManager.UUID(),
		authenticators:            clientManager.Authenticators(),
		options:                   options,
	}
	dispatcher, err := apiserverPlugins.GetPluginDispatcher(rm.k8sCoreClient, rm.runStore)
	if err != nil {
		glog.Errorf("failed to create plugin dispatcher: %s", err)
	}
	rm.pluginDispatcher = dispatcher
	return rm
}

func (r *ResourceManager) getWorkflowClient(namespace string) util.ExecutionInterface {
	return r.execClient.Execution(namespace)
}

func (r *ResourceManager) getScheduledWorkflowClient(namespace string) scheduledworkflowclient.ScheduledWorkflowInterface {
	return r.swfClient.ScheduledWorkflow(namespace)
}

// Creates a new experiment.
func (r *ResourceManager) CreateExperiment(experiment *model.Experiment) (*model.Experiment, error) {
	if common.IsMultiUserMode() {
		if experiment.Namespace == "" {
			return nil, util.NewInvalidInputError("Namespace cannot be empty")
		}
	}
	return r.experimentStore.CreateExperiment(experiment)
}

// Fetches an experiment with the given id.
func (r *ResourceManager) GetExperiment(experimentId string) (*model.Experiment, error) {
	return r.experimentStore.GetExperiment(experimentId)
}

// Fetches experiments with the given filtering and listing options.
func (r *ResourceManager) ListExperiments(filterContext *model.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error) {
	return r.experimentStore.ListExperiments(filterContext, opts)
}

// Deletes the experiment with the given id.
func (r *ResourceManager) DeleteExperiment(experimentId string) error {
	defaultExperimentId, err := r.GetDefaultExperimentId()
	if err != nil {
		return util.Wrapf(err, "Failed to delete experiment %v due to error fetching the default experiment id", experimentId)
	}
	if defaultExperimentId != "" && experimentId == defaultExperimentId {
		return util.NewBadRequestError(util.NewInvalidInputError("Experiment id cannot be equal to the default id %v", defaultExperimentId), "Failed to delete experiment %v. The default experiment cannot be deleted", experimentId)
	}
	if _, err := r.experimentStore.GetExperiment(experimentId); err != nil {
		return util.Wrapf(err, "Failed to delete experiment %v due to error fetching it", experimentId)
	}
	return r.experimentStore.DeleteExperiment(experimentId)
}

// Archives the experiment with the given id.
func (r *ResourceManager) ArchiveExperiment(ctx context.Context, experimentId string) error {
	// To archive an experiment
	// (1) update our persistent agent to disable CRDs of jobs in experiment
	// (2) update database to
	// (2.1) archive experiments
	// (2.2) archive runs
	// (2.3) disable jobs
	opts, err := list.NewOptions(&model.Job{}, 50, "name", nil)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to archive experiment %v", experimentId)
	}
	for {
		jobs, _, newToken, err := r.jobStore.ListJobs(&model.FilterContext{
			ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: experimentId},
		}, opts)
		if err != nil {
			return util.NewInternalServerError(err,
				"Failed to list jobs of to-be-archived experiment %v", experimentId)
		}
		for _, job := range jobs {
			k8sNamespace := job.Namespace
			if k8sNamespace == "" {
				k8sNamespace = common.GetPodNamespace()
			}
			_, err = r.getScheduledWorkflowClient(k8sNamespace).Patch(
				ctx,
				job.K8SName,
				types.MergePatchType,
				[]byte(fmt.Sprintf(`{"spec":{"enabled":%s}}`, strconv.FormatBool(false))))
			if err != nil {
				return util.NewInternalServerError(err,
					"Failed to disable job %v while archiving experiment %v", job.UUID, experimentId)
			}
		}
		if newToken == "" {
			break
		} else {
			opts, err = list.NewOptionsFromToken(newToken, 50)
			if err != nil {
				return util.NewInternalServerError(err,
					"Failed to create list jobs options from page token when archiving experiment %v", experimentId)
			}
		}
	}
	return r.experimentStore.ArchiveExperiment(experimentId)
}

// Un-archives the experiment with the given id.
func (r *ResourceManager) UnarchiveExperiment(experimentId string) error {
	return r.experimentStore.UnarchiveExperiment(experimentId)
}

// ListPipelines returns a list of pipelines. tagFilters is an optional map of tag key->value pairs for filtering.
func (r *ResourceManager) ListPipelines(filterContext *model.FilterContext, opts *list.Options, tagFilters map[string]string) ([]*model.Pipeline, int, string, error) {
	pipelines, totalSize, nextPageToken, err := r.pipelineStore.ListPipelines(filterContext, opts, tagFilters)
	if err != nil {
		err = util.Wrapf(err, "Failed to list pipelines with context %v, options %v", filterContext, opts)
	}
	return pipelines, totalSize, nextPageToken, err
}

// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
// Returns a list of pipelines using LEFT JOIN on SQL query.
// This could be more performant for a large number of pipeline versions.
func (r *ResourceManager) ListPipelinesV1(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, []*model.PipelineVersion, int, string, error) {
	pipelines, pipelineVersions, total_size, nextPageToken, err := r.pipelineStore.ListPipelinesV1(filterContext, opts)
	if err != nil {
		err = util.Wrapf(err, "ResourceManager (v1beta1): Failed to list pipelines with context %v, options %v", filterContext, opts)
	}
	return pipelines, pipelineVersions, total_size, nextPageToken, err
}

// Returns a pipeline.
func (r *ResourceManager) GetPipeline(pipelineId string) (*model.Pipeline, error) {
	if pipeline, err := r.pipelineStore.GetPipeline(pipelineId); err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline with id %v", pipelineId)
	} else {
		return pipeline, nil
	}
}

// Returns a pipeline specified by name and namespace.
func (r *ResourceManager) GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error) {
	if pipeline, err := r.pipelineStore.GetPipelineByNameAndNamespace(name, namespace); err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline named %v in namespace %v", name, namespace)
	} else {
		return pipeline, nil
	}
}

// TODO(gkcalat): consider removing after KFP v2 GA if users are not affected.
// Returns a pipeline specified by name and namespace using LEFT JOIN on SQL query.
// This could be more performant for a large number of pipeline versions.
func (r *ResourceManager) GetPipelineByNameAndNamespaceV1(name string, namespace string) (*model.Pipeline, *model.PipelineVersion, error) {
	if pipeline, pipelineVersion, err := r.pipelineStore.GetPipelineByNameAndNamespaceV1(name, namespace); err != nil {
		return nil, nil, util.Wrapf(err, "ResourceManager (v1beta1): Failed to get a pipeline named %v in namespace %v", name, namespace)
	} else {
		return pipeline, pipelineVersion, nil
	}
}

// Deletes a pipeline. Does not delete pipeline spec in the object storage.
// If cascade is false, fails if the pipeline has existing pipeline versions.
// If cascade is true, deletes all pipeline versions first, then deletes the pipeline.
func (r *ResourceManager) DeletePipeline(pipelineId string, cascade bool) error {
	// Check if pipeline exists
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return util.Wrapf(err, "Failed to delete pipeline with id %v as it was not found", pipelineId)
	}

	if cascade {
		// Get all pipeline versions for this pipeline and delete them
		opts := list.EmptyOptions()
		pipelineVersions, _, _, err := r.pipelineStore.ListPipelineVersions(pipelineId, opts, nil)
		if err != nil {
			return util.Wrapf(err, "Failed to delete pipeline with id %v due to error listing pipeline versions", pipelineId)
		}

		// Delete each pipeline version
		for _, pipelineVersion := range pipelineVersions {
			// Mark pipeline version as deleting so it's not visible to user.
			err = r.pipelineStore.UpdatePipelineVersionStatus(pipelineVersion.UUID, model.PipelineVersionDeleting)
			if err != nil {
				return util.Wrapf(err, "Failed to change the status of pipeline version id %v to DELETING during cascade delete", pipelineVersion.UUID)
			}

			// Delete the pipeline version from the database
			err = r.pipelineStore.DeletePipelineVersion(pipelineVersion.UUID)
			if err != nil {
				return util.Wrapf(err, "Failed to delete pipeline version %v during cascade delete of pipeline %v", pipelineVersion.UUID, pipelineId)
			}
			glog.Infof("Successfully deleted pipeline version %v during cascade delete of pipeline %v", pipelineVersion.UUID, pipelineId)
		}
	} else {
		// Check if it has no pipeline versions in Ready state
		latestPipelineVersion, err := r.pipelineStore.GetLatestPipelineVersion(pipelineId)
		if latestPipelineVersion != nil {
			return util.NewInvalidInputError("Failed to delete pipeline with id %v as it has existing pipeline versions (e.g. %v). Set cascade=true to delete all versions", pipelineId, latestPipelineVersion.UUID)
		} else if err.(*util.UserError).ExternalStatusCode() != codes.NotFound {
			return util.Wrapf(err, "Failed to delete pipeline with id %v as it failed to check existing pipeline versions", pipelineId)
		}
	}

	// Mark pipeline as deleting so it's not visible to user.
	err = r.pipelineStore.UpdatePipelineStatus(pipelineId, model.PipelineDeleting)
	if err != nil {
		return util.Wrapf(err, "Failed to change the status of pipeline id %v to DELETING", pipelineId)
	}

	// Delete a pipeline.
	err = r.pipelineStore.DeletePipeline(pipelineId)
	if err != nil {
		return util.Wrapf(err, "Failed to delete pipeline DB entry for pipeline id %v", pipelineId)
	}
	return nil
}

// TODO(gkcalat): consider removing before v2beta1 GA as default version is deprecated. This requires changes to v1beta1 proto.
// Updates default pipeline version for a given pipeline.
// Supports v1beta1 behavior.
func (r *ResourceManager) UpdatePipelineDefaultVersion(pipelineId string, versionId string) error {
	return r.pipelineStore.UpdatePipelineDefaultVersion(pipelineId, versionId)
}

// MaxTagKeyLength is the maximum allowed length (in characters) for a tag key.
// Consistent with Kubernetes label value length limit (63 characters).
const MaxTagKeyLength = model.MaxTagKeyLength

// MaxTagValueLength is the maximum allowed length (in characters) for a tag value.
// Consistent with Kubernetes label value length limit (63 characters).
const MaxTagValueLength = model.MaxTagValueLength

// MaxTagsPerEntity is the maximum number of tags allowed on a single pipeline or pipeline version.
const MaxTagsPerEntity = model.MaxTagsPerEntity

// UpdatePipeline updates mutable fields of a pipeline (display_name, tags).
// Both fields are updated in a single transaction via UpdatePipelineFields.
func (r *ResourceManager) UpdatePipeline(pipelineID string, displayName string, tags map[string]string) (*model.Pipeline, error) {
	if pipelineID == "" {
		return nil, util.NewInvalidInputError("pipeline id cannot be empty when updating pipeline")
	}
	if err := model.ValidateTags(tags); err != nil {
		return nil, err
	}
	// Update fields and tags in a single transaction to prevent deadlocks.
	if err := r.pipelineStore.UpdatePipelineFields(pipelineID, displayName, tags); err != nil {
		return nil, util.Wrap(err, "Failed to update pipeline")
	}
	// Return the updated pipeline.
	return r.pipelineStore.GetPipeline(pipelineID)
}

// UpdatePipelineVersion updates mutable fields of a pipeline version (display_name, tags)
// in a single transaction to prevent deadlocks.
func (r *ResourceManager) UpdatePipelineVersion(pipelineVersionID string, displayName string, tags map[string]string) (*model.PipelineVersion, error) {
	if pipelineVersionID == "" {
		return nil, util.NewInvalidInputError("pipeline version id cannot be empty when updating pipeline version")
	}
	if err := model.ValidateTags(tags); err != nil {
		return nil, err
	}
	// Update fields and tags in a single transaction to prevent deadlocks.
	if err := r.pipelineStore.UpdatePipelineVersionFields(pipelineVersionID, displayName, tags); err != nil {
		return nil, util.Wrap(err, "Failed to update pipeline version")
	}
	// Return the updated pipeline version.
	return r.pipelineStore.GetPipelineVersion(pipelineVersionID)
}

// Creates a pipeline, but does not create a pipeline version.
// Call CreatePipelineVersion to create a pipeline version.
func (r *ResourceManager) CreatePipeline(p *model.Pipeline) (*model.Pipeline, error) {
	if p.Name == "" {
		return nil, util.NewInvalidInputError("pipeline's name cannot be empty")
	}

	if p.DisplayName == "" {
		p.DisplayName = p.Name
	}

	if err := model.ValidateTags(p.Tags); err != nil {
		return nil, err
	}

	// Create a record in KFP DB (only pipelines table)
	newPipeline, err := r.pipelineStore.CreatePipeline(p)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline in PipelineStore")
	}

	newPipeline.Status = model.PipelineReady
	err = r.pipelineStore.UpdatePipelineStatus(
		newPipeline.UUID,
		newPipeline.Status,
	)
	if err != nil {
		return nil, util.Wrap(err, "Failed to update status of a pipeline after creation")
	}
	return newPipeline, nil
}

// Creates a pipeline and a pipeline version.
// This is used when two resources need to be created in a single DB transaction.
func (r *ResourceManager) CreatePipelineAndPipelineVersion(p *model.Pipeline, pv *model.PipelineVersion) (*model.Pipeline, *model.PipelineVersion, error) {
	if err := model.ValidateTags(p.Tags); err != nil {
		return nil, nil, err
	}
	if err := model.ValidateTags(pv.Tags); err != nil {
		return nil, nil, err
	}

	// Fetch pipeline spec, verify it, and parse parameters
	pipelineSpecBytes, pipelineSpecURI, err := r.fetchTemplateFromPipelineVersion(pv)
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to create a pipeline and a pipeline version as template is broken")
	}
	pv.PipelineSpec = model.LargeText(string(pipelineSpecBytes))
	if pipelineSpecURI != "" {
		pv.PipelineSpecURI = model.LargeText(pipelineSpecURI)
	}
	templateOptions := template.TemplateOptions{
		CacheDisabled:        r.options.CacheDisabled,
		DefaultWorkspace:     r.options.DefaultWorkspace,
		MLPipelineTLSEnabled: r.options.MLPipelineTLSEnabled,
		DefaultRunAsUser:     r.options.DefaultRunAsUser,
		DefaultRunAsGroup:    r.options.DefaultRunAsGroup,
		DefaultRunAsNonRoot:  r.options.DefaultRunAsNonRoot,
		DefaultHostUsers:     r.options.DefaultHostUsers,
	}
	tmpl, err := template.New(pipelineSpecBytes, templateOptions)
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to create a pipeline and a pipeline version due to template creation error")
	}
	if tmpl.GetTemplateType() == template.V1 {
		ns := p.Namespace
		if ns == "" {
			ns = common.GetPodNamespace()
		}
		if util.IsV1PipelinesBlocked(ns) {
			return nil, nil, util.NewInvalidInputError("V1 pipeline specs are not allowed. Please migrate to using KFP V2 pipelines.")
		}
	}
	// Validate pipeline's name in:
	// 1. pipeline spec for v2 pipelines and v2-compatible pipeline must comply with MLMD requirements
	// 2. display name must be non-empty
	pipelineSpecName := ""
	if tmpl.IsV2() {
		pipelineSpecName = tmpl.V2PipelineName()
		if err := common.ValidatePipelineName(pipelineSpecName); err != nil {
			return nil, nil, err
		}
	}
	if pv.Name == "" && p.Name == "" {
		if pipelineSpecName == "" {
			return nil, nil, util.NewInvalidInputError("pipeline's name cannot be empty")
		}
		pv.Name = pipelineSpecName
		p.Name = pipelineSpecName
	} else if pv.Name == "" {
		pv.Name = p.Name
	} else if p.Name == "" {
		p.Name = pv.Name
	}

	if pv.DisplayName == "" {
		pv.DisplayName = pv.Name
	}

	if p.DisplayName == "" {
		p.DisplayName = p.Name
	}

	// Parse parameters
	paramsJSON, err := tmpl.ParametersJSON()
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to create a pipeline and a pipeline version due to error converting parameters to json")
	}
	pv.Parameters = model.LargeText(paramsJSON)
	pv.PipelineSpec = model.LargeText(string(tmpl.Bytes()))

	// Create records in KFP DB (both pipelines and pipeline_versions tables)
	newPipeline, newVersion, err := r.pipelineStore.CreatePipelineAndPipelineVersion(p, pv)
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to create a pipeline and a pipeline version")
	}

	newPipeline.Status = model.PipelineReady
	err = r.pipelineStore.UpdatePipelineStatus(
		newPipeline.UUID,
		newPipeline.Status,
	)
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to update status of a new pipeline after creation")
	}
	newVersion.Status = model.PipelineVersionReady
	err = r.pipelineStore.UpdatePipelineVersionStatus(
		newVersion.UUID,
		newVersion.Status,
	)
	if err != nil {
		return nil, nil, util.Wrap(err, "Failed to update status of a new pipeline version after creation")
	}

	return newPipeline, newVersion, nil
}

// Updates the status of a pipeline.
func (r *ResourceManager) UpdatePipelineStatus(pipelineId string, status model.PipelineStatus) error {
	err := r.pipelineStore.UpdatePipelineStatus(pipelineId, status)
	if err != nil {
		return util.Wrapf(err, "Failed to update the status of pipeline id %v to %v", pipelineId, status)
	}
	return nil
}

// Updates the status of a pipeline version.
func (r *ResourceManager) UpdatePipelineVersionStatus(pipelineVersionId string, status model.PipelineVersionStatus) error {
	err := r.pipelineStore.UpdatePipelineVersionStatus(pipelineVersionId, status)
	if err != nil {
		return util.Wrapf(err, "Failed to update the status of pipeline version id %v to %v", pipelineVersionId, status)
	}
	return nil
}

// Returns the latest template for a specified pipeline id.
func (r *ResourceManager) GetPipelineLatestTemplate(pipelineId string) ([]byte, error) {
	// Verify pipeline exists
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get the latest template as pipeline was not found")
	}

	// Get the latest pipeline version
	latestPipelineVersion, err := r.pipelineStore.GetLatestPipelineVersion(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get the latest template for a pipeline")
	}

	// Fetch template []byte array
	if bytes, _, err := r.fetchTemplateFromPipelineVersion(latestPipelineVersion); err != nil {
		return nil, util.Wrapf(err, "Failed to get the latest template for pipeline with id %v", pipelineId)
	} else {
		return bytes, nil
	}
}

// Creates a run and schedule a workflow CR.
// Manifest's namespace gets overwritten with the run.Namespace.
// Creating a run from recurring run prioritizes recurring run's pipeline spec over the run's one.
func (r *ResourceManager) CreateRun(ctx context.Context, run *model.Run) (*model.Run, error) {
	// Guard against duplicate runs from concurrent recurring-run controller replicas.
	if run.RecurringRunId != "" && run.DisplayName != "" {
		existingRunID, err := r.runStore.GetRunByRecurringRunIDAndDisplayName(run.RecurringRunId, run.DisplayName)
		if err != nil {
			return nil, util.Wrap(err, "Failed to check for existing run")
		}
		if existingRunID != "" {
			return r.runStore.GetRun(existingRunID)
		}
	}

	// Create a template based on the manifest of an existing pipeline version or used-provided manifest.
	// Update the run.PipelineSpec if an existing pipeline version is used.
	tmpl, manifest, err := r.fetchTemplateFromPipelineSpec(&run.PipelineSpec)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a run due to error fetching manifest")
	}

	// TODO(gkcalat): consider changing the flow. Other resource UUIDs are assigned by their respective stores (DB).
	// Proposed flow:
	// 1. Create an entry and assign creation timestamp and uuid.
	// 2. Create a workflow CR.
	// 3. Update a record in the DB with scheduled timestamp, state, etc.
	// 4. Persistence agent will call apiserver to update the records later.
	if run.UUID == "" {
		// For runs created from a recurring run, derive a deterministic run ID from the
		// recurring run ID and display name. Concurrent triggers (e.g. multiple controller
		// replicas) then converge on the same primary key, so the second insert collides
		// and is resolved idempotently by the run store instead of creating a duplicate.
		if run.RecurringRunId != "" && run.DisplayName != "" {
			run.UUID = util.NewDeterministicUUID(run.RecurringRunId + "/" + run.DisplayName)
		} else {
			uuid, err := r.uuid.NewRandom()
			if err != nil {
				return nil, util.NewInternalServerError(err, "Failed to generate run ID")
			}
			run.UUID = uuid.String()
		}
	}
	run.RunDetails.CreatedAtInSec = r.time.Now().Unix()
	runWorkflowOptions := template.RunWorkflowOptions{
		RunID: run.UUID,
		RunAt: run.CreatedAtInSec,
	}
	executionSpec, err := tmpl.RunWorkflow(run, runWorkflowOptions)
	if err != nil {
		return nil, util.Wrap(err, "Failed to generate the ExecutionSpec")
	}
	err = executionSpec.Validate(false, false)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to validate workflow for (%+v)", executionSpec.ExecutionName())
	}
	// Create argo workflow CR resource
	k8sNamespace := run.Namespace
	if k8sNamespace == "" {
		k8sNamespace = common.GetPodNamespace()
	}
	if k8sNamespace == "" {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Namespace cannot be empty when creating an Argo workflow. Check if you have specified POD_NAMESPACE or try adding the parent namespace to the request"), "Failed to create a run due to empty namespace")
	}

	if util.IsV1PipelinesBlocked(k8sNamespace) && tmpl.GetTemplateType() == template.V1 {
		return nil, util.NewInvalidInputError("Namespace %s is not allowed to run v1 pipelines. Please migrate to using KFP V2 pipelines.", k8sNamespace)
	}

	executionSpec.SetExecutionNamespace(k8sNamespace)

	// assign OwnerReference and canonical labels to scheduledworkflow
	if run.RecurringRunId != "" {
		job, err := r.jobStore.GetJob(run.RecurringRunId)
		if err != nil {
			return nil, util.NewInternalServerError(util.NewInvalidInputError("RecurringRunId doesn't exist: %s", run.RecurringRunId), "Failed to create a run due to invalid recurring run id")
		}
		swf, err := r.swfClient.ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, v1.GetOptions{})
		if err != nil {
			return nil, util.NewInternalServerError(util.NewInvalidInputError("ScheduledWorkflow doesn't exist: %s", job.K8SName), "Failed to create a run due to invalid name")
		}
		executionSpec.SetOwnerReferences(swf)
		// canonical labels required for SWF controller label-based workflow tracking
		nextIndex := int64(1)
		if swf.Status.Trigger.LastIndex != nil {
			nextIndex = *swf.Status.Trigger.LastIndex + 1
		}
		executionSpec.SetCannonicalLabels(swf.Name, run.CreatedAtInSec, nextIndex)
	}

	if err := r.authorizeServiceAccount(ctx, executionSpec.ServiceAccount(), k8sNamespace); err != nil {
		return nil, util.Wrap(err, "Failed to create a run due to service account authorization error")
	}

	// Run plugin lifecycle hooks before workflow creation.
	pendingRun := &apiserverPlugins.PendingRun{
		RunID:             run.UUID,
		DisplayName:       run.DisplayName,
		Namespace:         k8sNamespace,
		PipelineID:        run.PipelineSpec.PipelineId,        //nolint:staticcheck // QF1008
		PipelineVersionID: run.PipelineSpec.PipelineVersionId, //nolint:staticcheck // QF1008
		PluginsInput:      (*string)(run.PluginsInputString),
	}
	if err := r.pluginDispatcher.OnBeforeRunCreation(ctx, pendingRun, executionSpec); err != nil {
		return nil, err
	}
	// Copy plugin output back to the model.
	if pendingRun.PluginsOutput != nil {
		lt := model.LargeText(*pendingRun.PluginsOutput)
		run.PluginsOutputString = &lt
	}

	runPersisted := false

	defer func() {
		if !runPersisted {
			if pr, prErr := apiserverPlugins.ModelToPersistedRun(run, k8sNamespace); prErr == nil {
				r.pluginDispatcher.OnRunEnd(ctx, pr)
			}
		}
	}()

	newExecSpec, err := r.getWorkflowClient(k8sNamespace).Create(ctx, executionSpec, v1.CreateOptions{})
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, util.NewUnavailableServerError(err, "Failed to create a workflow for (%s) - try again later", executionSpec.ExecutionName())
		}
		return nil, util.NewInternalServerError(err, "Failed to create a workflow for (%s)", executionSpec.ExecutionName())
	}
	// Update the run with the new scheduled workflow
	run.Namespace = k8sNamespace
	run.K8SName = newExecSpec.ExecutionName()
	run.ServiceAccount = newExecSpec.ServiceAccount()
	run.RunDetails.State = model.RuntimeState(string(newExecSpec.ExecutionStatus().Condition())).ToV2()
	run.RunDetails.Conditions = string(run.RunDetails.State.ToV1())
	// TODO(gkcalat): consider to avoid updating runtime manifest at create time and let
	// persistence agent update the runtime data.
	if tmpl.GetTemplateType() == template.V1 && run.RunDetails.WorkflowRuntimeManifest == "" {
		run.WorkflowRuntimeManifest = model.LargeText(newExecSpec.ToStringForStore())
		run.WorkflowSpecManifest = model.LargeText(manifest)
	} else if tmpl.GetTemplateType() == template.V2 {
		run.PipelineRuntimeManifest = model.LargeText(newExecSpec.ToStringForStore())
		run.PipelineSpecManifest = model.LargeText(manifest)
	} else {
		run.PipelineSpecManifest = model.LargeText(manifest)
	}
	// Assign the scheduled at time
	if run.RunDetails.ScheduledAtInSec == 0 {
		// if there is no scheduled time, then we assume this run is scheduled at the same time it is created
		run.RunDetails.ScheduledAtInSec = run.RunDetails.CreatedAtInSec
	}
	run.State = model.RuntimeStatePending

	newRun, err := r.runStore.CreateRun(run)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a run")
	}

	runPersisted = true

	// Upon run creation, update owning experiment
	err = r.experimentStore.SetLastRunTimestamp(newRun)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("Failed to set last run timestamp on experiment %s for run %s", newRun.ExperimentId, newRun.UUID))
	}

	return newRun, nil
}

// ReconcileSwfCrs reconciles the ScheduledWorkflow CRs based on existing jobs.
func (r *ResourceManager) ReconcileSwfCrs(ctx context.Context) error {
	filterContext := model.EmptyFilterContext()

	opts := list.EmptyOptions()

	jobs, _, _, err := r.jobStore.ListJobs(filterContext, opts)
	if err != nil {
		return util.Wrap(err, "Failed to reconcile ScheduledWorkflow Kubernetes resources")
	}

	for i := range jobs {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// If the pipeline isn't pinned, skip it. The runs API is used directly by the ScheduledWorkflow controller
		// in this case with just the pipeline ID and optionally the pipeline version ID.
		if jobs[i].PipelineSpec.PipelineSpecManifest == "" && jobs[i].PipelineSpec.WorkflowSpecManifest == "" {
			continue
		}

		tmpl, _, err := r.fetchTemplateFromPipelineSpec(&jobs[i].PipelineSpec)
		if err != nil {
			return failedToReconcileSwfCrsError(err)
		}

		newScheduledWorkflow, err := tmpl.ScheduledWorkflow(jobs[i])
		if err != nil {
			return failedToReconcileSwfCrsError(err)
		}

		for {
			currentScheduledWorkflow, err := r.getScheduledWorkflowClient(jobs[i].Namespace).Get(ctx, jobs[i].K8SName, v1.GetOptions{})
			if err != nil {
				if util.IsNotFound(err) {
					break
				}
				return failedToReconcileSwfCrsError(err)
			}

			if !reflect.DeepEqual(currentScheduledWorkflow.Spec, newScheduledWorkflow.Spec) {
				currentScheduledWorkflow.Spec = newScheduledWorkflow.Spec
				err = r.updateSwfCrSpec(ctx, jobs[i].Namespace, currentScheduledWorkflow)
				if err != nil {
					if apierrors.IsConflict(errors.Unwrap(err)) {
						continue
					} else if util.IsNotFound(errors.Cause(err)) {
						break
					}
					return failedToReconcileSwfCrsError(err)
				}
			}
			break
		}
	}

	return nil
}

func failedToReconcileSwfCrsError(err error) error {
	return util.Wrap(err, "Failed to reconcile ScheduledWorkflow Kubernetes resources")
}

func (r *ResourceManager) updateSwfCrSpec(ctx context.Context, k8sNamespace string, scheduledWorkflow *scheduledworkflow.ScheduledWorkflow) error {
	_, err := r.getScheduledWorkflowClient(k8sNamespace).Update(ctx, scheduledWorkflow)
	if err != nil {
		return util.Wrap(err, "Failed to update ScheduledWorkflow")
	}
	return nil
}

// Fetches a run with a given id.
func (r *ResourceManager) GetRun(runId string) (*model.Run, error) {
	run, err := r.runStore.GetRun(runId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to fetch run %v", runId)
	}
	return run, nil
}

// Fetches runs with a given set of filtering and listing options.
func (r *ResourceManager) ListRuns(filterContext *model.FilterContext, opts *list.Options) ([]*model.Run, int, string, error) {
	runs, totalSize, nextPageToken, err := r.runStore.ListRuns(filterContext, opts)
	if err != nil {
		return nil, 0, "", util.Wrap(err, "Failed to list runs")
	}
	return runs, totalSize, nextPageToken, nil
}

// Archives a run with a given id.
func (r *ResourceManager) ArchiveRun(runId string) error {
	if _, err := r.GetRun(runId); err != nil {
		return util.Wrapf(err, "Failed to archive run %v as it failed to be retrieved", runId)
	}
	if err := r.runStore.ArchiveRun(runId); err != nil {
		return util.Wrapf(err, "Failed to archive run %v", runId)
	}
	return nil
}

// Un-archives a run with a given id.
func (r *ResourceManager) UnarchiveRun(runId string) error {
	run, err := r.GetRun(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to unarchive run %v as it does not exist", runId)
	}
	if run.ExperimentId == "" {
		experimentRef, err := r.resourceReferenceStore.GetResourceReference(runId, model.RunResourceType, model.ExperimentResourceType)
		if err != nil {
			return util.Wrapf(err, "Failed to unarchive run %v due to resource references fetching error", runId)
		}
		run.ExperimentId = experimentRef.ReferenceUUID
	}

	experiment, err := r.GetExperiment(run.ExperimentId)
	if err != nil {
		return util.Wrapf(err, "Failed to unarchive run %v due to experiment fetching error", runId)
	}
	if experiment.StorageState.ToV2() == model.StorageStateArchived {
		return util.NewFailedPreconditionError(
			errors.New("Unarchive the experiment first to allow the run to be restored"),
			"%s", fmt.Sprintf("Failed to unarchive run %v as experiment %v must be un-archived first", runId, run.ExperimentId),
		)
	}
	if err := r.runStore.UnarchiveRun(runId); err != nil {
		return util.Wrapf(err, "Failed to unarchive run %v", runId)
	}
	return nil
}

// newStandardBackoffPolicy returns a configured backoff policy for retrying operations.
func newStandardBackoffPolicy() backoff.BackOff {
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.InitialInterval = 100 * time.Millisecond
	exponentialBackoff.MaxInterval = 5 * time.Second
	return backoff.WithMaxRetries(exponentialBackoff, 10)
}

// Deletes a run entry with a given id.
func (r *ResourceManager) DeleteRun(ctx context.Context, runId string) error {
	run, err := r.GetRun(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to delete run %v as it does not exist", runId)
	}
	if run.Namespace == "" {
		namespace, err := r.GetNamespaceFromExperimentId(run.ExperimentId)
		if err != nil {
			return util.Wrapf(err, "Failed to delete a run %v due to namespace fetching error", runId)
		}
		run.Namespace = namespace
	}
	k8sNamespace := run.Namespace
	if k8sNamespace == "" {
		k8sNamespace = common.GetPodNamespace()
	}
	err = r.getWorkflowClient(k8sNamespace).Delete(ctx, run.K8SName, v1.DeleteOptions{})
	if err != nil {
		// API won't need to delete the workflow CR
		// once persistent agent sync the state to DB and set TTL for it.
		glog.Warningf("Failed to delete run %v. Error: %v", run.K8SName, err.Error())
	}
	err = r.runStore.DeleteRun(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to delete a run %v", runId)
	}
	r.storedWorkflowIdentities.delete(runId)

	if r.options.CollectMetrics {
		if run.Conditions == string(exec.ExecutionSucceeded) {
			if util.GetMetricValue(workflowSuccessCounter) > 0 {
				workflowSuccessCounter.WithLabelValues(run.Namespace, run.DisplayName).Dec()
			}
		} else {
			if util.GetMetricValue(workflowFailedCounter) > 0 {
				workflowFailedCounter.WithLabelValues(run.Namespace, run.DisplayName).Dec()
			}
		}
	}
	return nil
}

// Creates a task entry.
func (r *ResourceManager) CreateTask(t *model.Task) (*model.Task, error) {
	run, err := r.GetRun(t.RunID)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a task for run %v", t.RunID)
	}
	if run.ExperimentId == "" {
		defaultExperimentId, err := r.GetDefaultExperimentId()
		if err != nil {
			return nil, util.Wrapf(err, "Failed to create a task in run %v. Specify experiment id for the run or check if the default experiment exists", t.RunID)
		}
		run.ExperimentId = defaultExperimentId
	}

	// Validate namespace
	if t.Namespace == "" {
		namespace, err := r.GetNamespaceFromExperimentId(run.ExperimentId)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to create a task in run %v", t.RunID)
		}
		t.Namespace = namespace
	}
	if common.IsMultiUserMode() {
		if t.Namespace == "" {
			return nil, util.NewInternalServerError(util.NewInvalidInputError("Task cannot have an empty namespace in multi-user mode"), "Failed to create a task in run %v", t.RunID)
		}
	}
	if err := r.CheckExperimentBelongsToNamespace(run.ExperimentId, t.Namespace); err != nil {
		return nil, util.Wrapf(err, "Failed to create a task in run %v", t.RunID)
	}

	newTask, err := r.taskStore.CreateTask(t)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a task in run %v", t.RunID)
	}
	return newTask, nil
}

// Fetches tasks with a given set of filtering and listing options.
func (r *ResourceManager) ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error) {
	tasks, totalSize, nextPageToken, err := r.taskStore.ListTasks(filterContext, opts)
	if err != nil {
		return nil, 0, "", util.Wrap(err, "Failed to list tasks")
	}
	return tasks, totalSize, nextPageToken, nil
}

// Fetches recurring runs with given filtering and listing options.
func (r *ResourceManager) ListJobs(filterContext *model.FilterContext, opts *list.Options) ([]*model.Job, int, string, error) {
	return r.jobStore.ListJobs(filterContext, opts)
}

// Terminates a workflow by setting its activeDeadlineSeconds to 0.
func TerminateWorkflow(ctx context.Context, wfClient util.ExecutionInterface, name string) error {
	patchObj := util.GetTerminatePatch(util.CurrentExecutionType())
	patch, err := json.Marshal(patchObj)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to terminate workflow %s due to error parsing the patch", name)
	}
	operation := func() error {
		_, err = wfClient.Patch(ctx, name, types.MergePatchType, patch, v1.PatchOptions{})
		return util.Wrapf(err, "Failed to terminate workflow %s due to patching error", name)
	}
	err = backoff.Retry(operation, newStandardBackoffPolicy())
	if err != nil {
		return util.Wrapf(err, "Failed to terminate workflow %s due to patching error after multiple retries", name)
	}
	return nil
}

// Terminates a running run and the corresponding workflow.
func (r *ResourceManager) TerminateRun(ctx context.Context, runId string) error {
	run, err := r.GetRun(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to terminate run %s due to error fetching the run", runId)
	}
	namespace, err := r.getNamespaceFromRunId(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to terminate run %s due to error fetching its namespace", runId)
	}

	err = r.runStore.TerminateRun(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to terminate run %s", runId)
	}

	if namespace == "" {
		namespace = common.GetPodNamespace()
	}
	err = TerminateWorkflow(ctx, r.getWorkflowClient(namespace), run.K8SName)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to terminate run %s due to error terminating its workflow", runId)
	}
	return nil
}

// Retries a run given its id.
func (r *ResourceManager) RetryRun(ctx context.Context, runId string) error {
	run, err := r.GetRun(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to retry run %s due to error fetching the run", runId)
	}
	if run.StorageState.ToV2() == model.StorageStateArchived {
		return storage.NewArchivedRunRetryError(runId)
	}
	// TODO(gkcalat): consider using run.Namespace after migration logic will be available.
	namespace, err := r.getNamespaceFromRunId(runId)
	if err != nil {
		return util.Wrapf(err, "Failed to retry run %s due to error fetching its namespace", runId)
	}

	if run.RunDetails.WorkflowRuntimeManifest == "" {
		return util.NewBadRequestError(util.NewInvalidInputError("Workflow manifest cannot be empty"), "Failed to retry run %s due to error fetching workflow manifest", runId)
	}
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.RunDetails.WorkflowRuntimeManifest))
	if err != nil {
		return util.NewInternalServerError(err, "Failed to retry run %s due to error parsing the workflow manifest", runId)
	}

	if err := execSpec.Decompress(); err != nil {
		return util.NewInternalServerError(err, "Failed to retry run %s due to error decompressing execution spec", runId)
	}

	if err := execSpec.CanRetry(); err != nil {
		return util.NewInternalServerError(err, "Failed to retry run %s as it does not allow retries", runId)
	}

	newExecSpec, podsToDelete, err := execSpec.GenerateRetryExecution()
	if err != nil {
		return util.Wrapf(err, "Failed to retry run %s", runId)
	}

	if namespace == "" {
		namespace = common.GetPodNamespace()
	}

	// If a previous retry claim has aged out, reconcile against Kubernetes
	// before deciding anything: an expired claim is not necessarily an
	// abandoned one. If the claim's workflow exists (the previous API server
	// crashed after applying the mutation but before persisting), adopt it
	// instead of launching a duplicate; only a definitive absence (NotFound,
	// or a live workflow still carrying an older generation) authorizes the
	// takeover below.
	allowClaimTakeover := false
	if run.State == model.RuntimeStatePending && run.RetryGeneration > 0 {
		claimAge := r.time.Now().Unix() - run.RetryClaimedAtInSec
		if run.RetryClaimedAtInSec == 0 || claimAge > int64(retryClaimGracePeriod()/time.Second) {
			liveWorkflow, readError := r.getWorkflowClient(namespace).Get(ctx, execSpec.ExecutionName(), v1.GetOptions{})
			switch {
			case readError == nil && liveWorkflow != nil &&
				reportedRetryGeneration(liveWorkflow.ExecutionObjectMeta()) == run.RetryGeneration:
				// The previous retry was applied. Persist its current state
				// and report success: the retry the user asked for is
				// already running (or finished).
				glog.Warningf("Run %s has an expired retry claim (generation %d) but its workflow is live; adopting it instead of retrying again", runId, run.RetryGeneration)
				condition := string(liveWorkflow.ExecutionStatus().Condition())
				run.Conditions = condition
				run.State = model.RuntimeState(condition).ToV2()
				run.FinishedAtInSec = liveWorkflow.ExecutionStatus().FinishedAt()
				run.WorkflowRuntimeManifest = model.LargeText(liveWorkflow.ToStringForStore())
				run.K8SName = liveWorkflow.ExecutionName()
				// The crashed retry may not have reached plugin
				// notification, so adoption fires it (mirrors the normal
				// retry path); delivery is documented as at-least-once and
				// handlers deduplicate on (RunID, RetryGeneration).
				if run.PluginsOutputString != nil && *run.PluginsOutputString != "" {
					if pr, prErr := apiserverPlugins.ModelToPersistedRun(run, namespace); prErr == nil {
						r.pluginDispatcher.OnRunRetry(ctx, pr)
					}
				}
				run.PluginsOutputString = nil
				if updateError := r.runStore.UpdateRun(run); updateError != nil {
					return util.NewInternalServerError(updateError, "Failed to adopt in-flight retry for run %s", runId)
				}
				r.storedWorkflowIdentities.delete(runId)
				return nil
			case readError != nil && !apierrors.IsNotFound(readError):
				// Transient read: preserve the claim rather than risking a
				// takeover that duplicates live work.
				return util.NewUnavailableServerError(readError,
					"Run %s has an expired retry claim but its workflow state could not be verified - try again later", runId)
			default:
				allowClaimTakeover = true
			}
		}
	}

	// Atomically claim via database-side CAS to prevent ReportWorkflowResource
	// from overwriting with a stale terminal state. The returned claimGeneration
	// acts as a unique fence token: UpdateRun checks it to reject stale reports,
	// and RollbackRetryClaim checks it to prevent ABA rollback of a later retry.
	originalState, originalConditions, originalFinishedAtInSec, claimGeneration, claimError := r.runStore.ClaimRunForRetry(runId, allowClaimTakeover)
	if claimError != nil {
		// Wrap (not re-classify) so NotFound / BadRequest from the claim
		// reach the client as such instead of surfacing as HTTP 500.
		return util.Wrapf(claimError,
			"Failed to retry run %s: could not claim database row before workflow operation", runId)
	}
	// Update the in-memory run to reflect the claimed state.
	run.FinishedAtInSec = 0
	run.State = model.RuntimeStatePending
	run.Conditions = string(model.RuntimeStatePending.ToV1())
	run.RetryGeneration = claimGeneration
	run.RetryClaimedAtInSec = r.time.Now().Unix()
	// Stamp the claim token on the workflow so ReportWorkflowResource can
	// distinguish reports about this retried workflow from stale snapshots
	// of the pre-retry workflow, without comparing timestamps across clocks.
	newExecSpec.SetAnnotations(util.AnnotationKeyRetryGeneration, strconv.FormatInt(claimGeneration, 10))

	if err = deletePods(ctx, r.k8sCoreClient, podsToDelete, namespace); err != nil {
		// Pod deletion is a local operation that precedes any workflow mutation.
		// Safe to rollback unconditionally — no external state was changed.
		if rollbackError := r.runStore.RollbackRetryClaim(runId, originalState, originalConditions, originalFinishedAtInSec, claimGeneration); rollbackError != nil {
			glog.Errorf("Failed to rollback retry claim for run %s after pod deletion failure: %v", runId, rollbackError)
		}
		return util.NewInternalServerError(err, "Failed to retry run %s due to error cleaning up the failed pods from the previous attempt", runId)
	}

	// Capture the workflow name the retry operates on before newExecSpec is
	// reassigned (it is nil on error). This is the name from the stored
	// runtime manifest, which is the object updateOrCreateRetryWorkflow
	// mutates; run.K8SName can diverge from it.
	retryWorkflowName := newExecSpec.ExecutionName()
	newExecSpec, err = r.updateOrCreateRetryWorkflow(ctx, namespace, runId, newExecSpec)
	if err != nil {
		// Workflow reconciliation failed. Kubernetes timeouts and 5xx responses
		// are ambiguous: the API server may have applied the running workflow
		// even after the client exhausted retries. Re-read the live workflow
		// to determine whether the mutation was applied.
		workflowClient := r.getWorkflowClient(namespace)
		liveWorkflow, readError := workflowClient.Get(ctx, retryWorkflowName, v1.GetOptions{})
		switch {
		case readError == nil && liveWorkflow != nil &&
			reportedRetryGeneration(liveWorkflow.ExecutionObjectMeta()) == claimGeneration:
			// The mutation was applied despite the error: the live workflow
			// carries this claim's generation (it may even be terminal
			// already if the retry finished quickly). Adopt it and complete
			// the retry instead of rolling back — a rollback here would
			// restore a GC-eligible FinishedAtInSec under a live retried
			// workflow and permit a duplicate retry.
			glog.Warningf("Retry workflow for run %s returned error but the live workflow carries claim generation %d; adopting it. Original error: %v",
				runId, claimGeneration, err)
			newExecSpec = liveWorkflow
		case readError == nil && liveWorkflow != nil && !liveWorkflow.ExecutionStatus().IsInFinalState():
			// Workflow exists and is running — mutation was applied.
			// Preserve the claimed row for reconciliation.
			glog.Warningf("Retry workflow for run %s returned error but workflow is live (not terminal). "+
				"Preserving claimed row for reconciliation. Original error: %v", runId, err)
			return util.NewUnavailableServerError(err,
				"Retry workflow for run %s returned error but workflow is live; claim preserved for reconciliation", runId)
		case readError != nil && !apierrors.IsNotFound(readError):
			// Ambiguous: GET itself failed with a transient error. The
			// workflow may be running despite the read failure. Preserve
			// the claimed row for reconciliation rather than risking
			// rollback to a GC-eligible timestamp.
			glog.Warningf("Retry workflow for run %s failed and live workflow read also failed; "+
				"preserving claimed row for reconciliation. Workflow error: %v, read error: %v", runId, err, readError)
			return util.NewUnavailableServerError(err,
				"Retry workflow for run %s failed with ambiguous state; claim preserved for reconciliation", runId)
		default:
			// Workflow definitively absent (NotFound), or terminal without
			// this claim's generation — a pre-retry leftover, so the
			// mutation was provably not applied. Safe to rollback.
			if rollbackError := r.runStore.RollbackRetryClaim(runId, originalState, originalConditions, originalFinishedAtInSec, claimGeneration); rollbackError != nil {
				glog.Errorf("Failed to rollback retry claim for run %s after workflow reconciliation failure: %v", runId, rollbackError)
			}
			return err
		}
	}
	// Notify plugins of retry
	if run.PluginsOutputString != nil && *run.PluginsOutputString != "" {
		if pr, prErr := apiserverPlugins.ModelToPersistedRun(run, namespace); prErr == nil {
			r.pluginDispatcher.OnRunRetry(ctx, pr)
		}
	}

	condition := string(newExecSpec.ExecutionStatus().Condition())
	run.Conditions = condition
	// 0 for a freshly resubmitted (running) workflow; the real finish time
	// when the reconciliation path adopted an already-terminal retry.
	run.FinishedAtInSec = newExecSpec.ExecutionStatus().FinishedAt()
	run.WorkflowRuntimeManifest = model.LargeText(newExecSpec.ToStringForStore())
	run.K8SName = newExecSpec.ExecutionName()
	run.State = model.RuntimeState(condition).ToV2()
	// OnRunRetry persists plugin output independently; leave PluginsOutput unchanged here.
	run.PluginsOutputString = nil
	err = r.runStore.UpdateRun(run)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to retry run %s due to error updating entry", runId)
	}
	r.storedWorkflowIdentities.delete(runId)
	return nil
}

func (r *ResourceManager) updateOrCreateRetryWorkflow(ctx context.Context, namespace string, runID string, newExecSpec util.ExecutionSpec) (util.ExecutionSpec, error) {
	workflowClient := r.getWorkflowClient(namespace)
	var retriedWorkflow util.ExecutionSpec
	var lastWorkflowError error
	lastWorkflowAction := "reconciling workflow"

	err := retry.OnError(retry.DefaultRetry, isRetryableWorkflowReconcileError, func() error {
		lastWorkflowAction = "getting workflow"
		latestWorkflow, err := workflowClient.Get(ctx, newExecSpec.ExecutionName(), v1.GetOptions{})
		if err == nil {
			newExecSpec.SetVersion(latestWorkflow.Version())
			lastWorkflowAction = "updating workflow"
			updatedWorkflow, err := workflowClient.Update(ctx, newExecSpec, v1.UpdateOptions{})
			if err == nil {
				retriedWorkflow = updatedWorkflow
				return nil
			}
			lastWorkflowError = err
			if !apierrors.IsNotFound(err) {
				return err
			}
		} else {
			lastWorkflowError = err
			if !apierrors.IsNotFound(err) && !isTransientWorkflowReconcileError(err) {
				return err
			}
		}

		newExecSpec.SetVersion("")
		lastWorkflowAction = "creating workflow"
		newCreatedWorkflow, createError := workflowClient.Create(ctx, newExecSpec, v1.CreateOptions{})
		if createError == nil {
			retriedWorkflow = newCreatedWorkflow
			return nil
		}
		lastWorkflowError = createError
		return createError
	})
	if err == nil {
		return retriedWorkflow, nil
	}

	lastWorkflowErrorMessage := "none"
	if lastWorkflowError != nil {
		lastWorkflowErrorMessage = lastWorkflowError.Error()
	}
	if apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) {
		return nil, util.NewUnavailableServerError(err, "Failed to retry run %s due to error reconciling workflow after retries - try again later. Last workflow error: %s", runID, lastWorkflowErrorMessage)
	}
	if isTransientWorkflowReconcileError(err) {
		return nil, util.NewUnavailableServerError(err, "Failed to retry run %s due to error %s - try again later. Last workflow error: %s", runID, lastWorkflowAction, lastWorkflowErrorMessage)
	}
	return nil, util.NewInternalServerError(err, "Failed to retry run %s due to error %s. Last workflow error: %s", runID, lastWorkflowAction, lastWorkflowErrorMessage)
}

func isRetryableWorkflowReconcileError(err error) bool {
	return apierrors.IsConflict(err) ||
		apierrors.IsAlreadyExists(err) ||
		isTransientWorkflowReconcileError(err)
}

func isTransientWorkflowReconcileError(err error) bool {
	if err == nil {
		return false
	}
	if netError, ok := err.(net.Error); ok && netError.Timeout() {
		return true
	}
	return apierrors.IsServerTimeout(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsInternalError(err) ||
		apierrors.IsUnexpectedServerError(err)
}

// Fetches execution logs and writes to the destination.
// 1. Attempts to read logs directly from pod.
// 2. Attempts to read logs from archive if reading from pod fails.
func (r *ResourceManager) ReadLog(ctx context.Context, runId string, nodeId string, follow bool, dst io.Writer) error {
	run, err := r.GetRun(runId)
	if err != nil {
		return util.NewBadRequestError(err, "Failed to read logs for run %v due to run fetching error", runId)
	}
	namespace, err := r.getNamespaceFromRunId(runId)
	if err != nil {
		return util.NewBadRequestError(err, "Failed to read logs for run %v due to namespace fetching error", runId)
	}
	err = r.readRunLogFromPod(ctx, runId, namespace, nodeId, follow, dst)
	if err != nil && r.logArchive != nil {
		err = r.readRunLogFromArchive(ctx, string(run.WorkflowRuntimeManifest), nodeId, dst)
		if err != nil {
			return util.NewBadRequestError(err, "Failed to read logs for run %v", runId)
		}
	}
	if err != nil {
		return util.NewBadRequestError(err, "Failed to read logs for run %v", runId)
	}
	return nil
}

// Fetches execution logs from a pod.
func (r *ResourceManager) readRunLogFromPod(ctx context.Context, runID string, namespace string, nodeID string, follow bool, dst io.Writer) error {
	// The caller controls nodeID, so confirm the pod was created by this run
	// before streaming, otherwise the run only selects a namespace and any pod
	// in it could be read with the API server's credentials.
	pod, err := r.k8sCoreClient.PodClient(namespace).Get(ctx, nodeID, v1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			glog.Errorf("Failed to get pod %v: %v", nodeID, err)
		}
		return util.NewInternalServerError(err, "Failed to read logs from pod %v due to error fetching the pod", nodeID)
	}
	if pod == nil || pod.Labels[util.LabelKeyWorkflowRunId] != runID {
		return util.NewInvalidInputError("Pod %v does not belong to run %v", nodeID, runID)
	}

	logOptions := corev1.PodLogOptions{
		Container:  "main",
		Timestamps: false,
		Follow:     follow,
	}

	req := r.k8sCoreClient.PodClient(namespace).GetLogs(nodeID, &logOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			glog.Errorf("Failed to read logs from pod %v: %v", nodeID, err)
		}
		return util.NewInternalServerError(err, "Failed to read logs from pod %v due to error opening log stream", nodeID)
	}
	defer podLogs.Close()

	_, err = io.Copy(dst, podLogs)
	if err != nil && !errors.Is(err, io.EOF) {
		return util.NewInternalServerError(err, "Failed to read logs from pod %v due to error in streaming the log", nodeID)
	}
	return nil
}

// Fetches execution logs from a archived pod logs.
func (r *ResourceManager) readRunLogFromArchive(ctx context.Context, workflowManifest string, nodeID string, dst io.Writer) error {
	if workflowManifest == "" {
		return util.NewInternalServerError(util.NewInvalidInputError("Runtime workflow manifest cannot empty"), "Failed to read logs from archive %v due to empty runtime workflow manifest", nodeID)
	}

	execSpec, err := util.NewExecutionSpecJSON(util.CurrentExecutionType(), []byte(workflowManifest))
	if err != nil {
		return util.NewInternalServerError(err, "Failed to read logs from archive %v due error reading execution spec", nodeID)
	}

	logPath, err := r.logArchive.GetLogObjectKey(execSpec, nodeID)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to read logs from archive %v", nodeID)
	}

	logReader, err := r.objectStore.GetFileReader(ctx, logPath)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to read logs from archive %v due to error fetching the log file", nodeID)
	}
	defer logReader.Close()

	err = r.logArchive.CopyLogFromArchiveReader(logReader, dst, archive.ExtractLogOptions{LogFormat: archive.LogFormatText, Timestamps: false})
	if err != nil {
		return util.NewInternalServerError(err, "Failed to read logs from archive %v due to error copying the log file", nodeID)
	}
	return nil
}

// Fetches a recurring run with given id.
func (r *ResourceManager) GetJob(id string) (*model.Job, error) {
	return r.jobStore.GetJob(id)
}

// Fetches or creates a new pipeline version based on internal PipelineSpec representation.
// Returns a pipeline version if any of the following is present in pipeline spec:
// 1. Pipeline version with the given pipeline version id
// 2. The latest pipeline version with given pipeline id
// 3. Repeats 1 and 2 for pipeline version id and pipeline id parsed from the pipeline name
func (r *ResourceManager) fetchPipelineVersionFromPipelineSpec(pipelineSpec model.PipelineSpec) (*model.PipelineVersion, error) {
	// Fetch or create a pipeline version
	if pipelineSpec.PipelineVersionId != "" {
		pipelineVersion, err := r.GetPipelineVersion(pipelineSpec.PipelineVersionId)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to fetch a pipeline version and its manifest from pipeline version %v", pipelineSpec.PipelineVersionId)
		}
		// Requests in v1beta1 may have empty pipeline ID. Therefore, we only catch
		// v2beta1 calls to create a run or recurring run with inconsistent pipeline ID.
		if pipelineVersion.PipelineId != "" && pipelineSpec.PipelineId != "" && pipelineVersion.PipelineId != pipelineSpec.PipelineId {
			return nil, util.NewInvalidInputError("Pipeline version %v belongs to pipeline %v (not %v)", pipelineSpec.PipelineVersionId, pipelineVersion.PipelineId, pipelineSpec.PipelineId)
		}
		return pipelineVersion, nil
	} else if pipelineSpec.PipelineId != "" {
		pipelineVersion, err := r.GetLatestPipelineVersion(pipelineSpec.PipelineId)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to fetch a pipeline version and its manifest from pipeline %v", pipelineSpec.PipelineId)
		}
		return pipelineVersion, nil
	} else if pipelineSpec.PipelineName != "" {
		resourceNames := common.ParseResourceIdsFromFullName(pipelineSpec.PipelineName)
		if resourceNames["PipelineVersionId"] == "" && resourceNames["PipelineId"] == "" {
			return nil, util.Wrapf(util.NewInvalidInputError("Pipeline spec source is missing"), "Failed to fetch a pipeline version and its manifest due to an empty pipeline spec source: %v", pipelineSpec.PipelineName)
		}
		if resourceNames["PipelineVersionId"] != "" {
			pipelineVersion, err := r.GetPipelineVersion(resourceNames["PipelineVersionId"])
			if err != nil {
				return nil, util.Wrapf(err, "Failed to fetch a pipeline version and its manifest from pipeline %v. Check if pipeline version %v exists", pipelineSpec.PipelineName, resourceNames["PipelineVersionId"])
			}
			return pipelineVersion, nil
		} else {
			pipelineVersion, err := r.GetLatestPipelineVersion(resourceNames["PipelineId"])
			if err != nil {
				return nil, util.Wrapf(err, "Failed to fetch a pipeline version and its manifest from pipeline %v. Check if pipeline %v exists", pipelineSpec.PipelineName, resourceNames["PipelineId"])
			}
			return pipelineVersion, nil
		}
	}
	return nil, nil
}

// Creates a recurring run.
// Manifest's namespace gets overwritten with the job.Namespace if the later is non-empty.
// Otherwise, job.Namespace gets overwritten by the manifest.
func (r *ResourceManager) CreateJob(ctx context.Context, job *model.Job) (*model.Job, error) {
	// Create a new ScheduledWorkflow at the ScheduledWorkflow client.
	k8sNamespace := job.Namespace
	if k8sNamespace == "" {
		k8sNamespace = common.GetPodNamespace()
	}
	if k8sNamespace == "" {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Namespace cannot be empty when creating an Argo scheduled workflow. Check if you have specified POD_NAMESPACE or try adding the parent namespace to the request"), "Failed to create a recurring run due to empty namespace")
	}

	job.Namespace = k8sNamespace

	var manifest string
	var scheduledWorkflow *scheduledworkflow.ScheduledWorkflow
	var tmpl template.Template

	// If the pipeline version or pipeline spec is provided, this means the user wants to pin to a specific pipeline.
	// Otherwise, always let the ScheduledWorkflow controller pick the latest.
	if job.PipelineVersionId != "" || job.PipelineSpecManifest != "" || job.WorkflowSpecManifest != "" {
		var err error
		// Create a template based on the manifest of an existing pipeline version or used-provided manifest.
		// Update the job.PipelineSpec if an existing pipeline version is used.
		tmpl, manifest, err = r.fetchTemplateFromPipelineSpec(&job.PipelineSpec)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to create a recurring run with an invalid pipeline spec manifest")
		}

		// When plugins are enabled, the SWF controller must call the CreateRun API
		// so that per-run plugin logic executes.
		if r.pluginDispatcher.PluginsRegistered() {
			// Plugin-enabled: create a lightweight SWF without inline workflow spec
			// so the SWF controller calls the CreateRun API for per-run plugin logic.
			scheduledWorkflow, err = template.NewGenericScheduledWorkflow(job)
		} else {
			// TODO(gkcalat): consider changing the flow. Other resource UUIDs are assigned by their respective stores (DB).
			// Convert modelJob into scheduledWorkflow.
			scheduledWorkflow, err = tmpl.ScheduledWorkflow(job)
		}
		if err != nil {
			return nil, util.Wrap(err, "Failed to create a recurring run during scheduled workflow creation")
		}
	} else if job.PipelineId == "" {
		return nil, errors.New("Cannot create a job with an empty pipeline ID")
	} else {
		// Validate the input parameters on the latest pipeline version. The latest pipeline version is not stored
		// in the ScheduledWorkflow. It's just to help the user with up front validation at recurring run creation
		// time.
		manifest, err := r.GetPipelineLatestTemplate(job.PipelineId)
		if err != nil {
			return nil, util.Wrap(err, "Failed to validate the input parameters on the latest pipeline version")
		}

		templateOptions := template.TemplateOptions{
			CacheDisabled:        r.options.CacheDisabled,
			DefaultWorkspace:     r.options.DefaultWorkspace,
			MLPipelineTLSEnabled: r.options.MLPipelineTLSEnabled,
			DefaultRunAsUser:     r.options.DefaultRunAsUser,
			DefaultRunAsGroup:    r.options.DefaultRunAsGroup,
			DefaultRunAsNonRoot:  r.options.DefaultRunAsNonRoot,
			DefaultHostUsers:     r.options.DefaultHostUsers,
		}
		tmpl, err := template.New(manifest, templateOptions)
		if err != nil {
			return nil, util.Wrap(err, "Failed to fetch a template with an invalid pipeline spec manifest")
		}

		validatedScheduledWorkflow, err := tmpl.ScheduledWorkflow(job)
		if err != nil {
			return nil, util.Wrap(err, "Failed to validate the input parameters on the latest pipeline version")
		}
		if v2Tmpl, ok := tmpl.(*template.V2Spec); ok {
			if err = v2Tmpl.ValidateJobInputs(job); err != nil {
				return nil, util.Wrap(err, "Failed to validate the input parameters on the latest pipeline version")
			}
		}

		scheduledWorkflow, err = template.NewGenericScheduledWorkflow(job)
		if err != nil {
			return nil, util.Wrap(err, "Failed to create a recurring run during scheduled workflow creation")
		}

		parameters, err := template.StringMapToCRDParameters(string(job.RuntimeConfig.Parameters))
		if err != nil {
			return nil, util.Wrap(err, "Converting runtime config's parameters to CDR parameters failed")
		}

		scheduledWorkflow.Spec.Workflow = &scheduledworkflow.WorkflowResource{
			Parameters: parameters, PipelineRoot: string(job.PipelineRoot),
		}
		scheduledWorkflow.Spec.ServiceAccount = validatedScheduledWorkflow.Spec.ServiceAccount
	}

	if tmpl != nil && util.IsV1PipelinesBlocked(k8sNamespace) && tmpl.GetTemplateType() == template.V1 {
		return nil, util.NewInvalidInputError("Namespace %s is not allowed to run v1 pipelines. Please migrate to using KFP V2 pipelines.", k8sNamespace)
	}

	resolvedJobServiceAccount := scheduledWorkflow.Spec.ServiceAccount
	if resolvedJobServiceAccount == "" {
		resolvedJobServiceAccount = job.ServiceAccount
	}
	if err := r.authorizeServiceAccount(ctx, resolvedJobServiceAccount, k8sNamespace); err != nil {
		return nil, util.Wrap(err, "Failed to create a recurring run due to service account authorization error")
	}

	newScheduledWorkflow, err := r.getScheduledWorkflowClient(k8sNamespace).Create(ctx, scheduledWorkflow)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, util.NewUnavailableServerError(err, "Failed to create a recurring run during scheduling a workflow - try again later")
		}
		return nil, util.Wrap(err, "Failed to create a recurring run during scheduling a workflow")
	}
	// Complete modelJob with info coming back from ScheduledWorkflow client.
	swf := util.NewScheduledWorkflow(newScheduledWorkflow)
	job.UUID = string(swf.UID)
	job.K8SName = swf.Name
	job.Conditions = model.StatusState(swf.ConditionSummary()).ToString()
	for _, modelRef := range job.ResourceReferences {
		modelRef.ResourceUUID = string(swf.UID)
	}

	if tmpl == nil {
		return r.jobStore.CreateJob(job)
	}

	if tmpl.GetTemplateType() == template.V1 {
		// Get the service account
		serviceAccount := ""
		if swf.Spec.Workflow != nil {
			execSpec, err := util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, swf.Spec.Workflow)
			if err == nil {
				serviceAccount = execSpec.ServiceAccount()
			}
		}
		job.ServiceAccount = serviceAccount
		job.WorkflowSpecManifest = model.LargeText(manifest)
	} else {
		job.ServiceAccount = newScheduledWorkflow.Spec.ServiceAccount
		job.PipelineSpecManifest = model.LargeText(manifest)
	}
	return r.jobStore.CreateJob(job)
}

// Enables or disables a recurring run with given id.
func (r *ResourceManager) ChangeJobMode(ctx context.Context, jobId string, enable bool) error {
	job, err := r.GetJob(jobId)
	if err != nil {
		return util.Wrapf(err, "Failed to change recurring run's mode to enable:%v. Check if recurring run %v exists", enable, jobId)
	}
	k8sNamespace := job.Namespace
	if k8sNamespace == "" {
		k8sNamespace = common.GetPodNamespace()
	}
	if enable {
		scheduledWorkflow, err := r.getScheduledWorkflowClient(k8sNamespace).Get(ctx, job.K8SName, v1.GetOptions{})
		if err != nil {
			return util.NewInternalServerError(err, "Failed to enable recurring run %v. Check if the scheduled workflow exists", jobId)
		}
		if scheduledWorkflow == nil || string(scheduledWorkflow.UID) != jobId {
			return util.Wrapf(util.NewResourceNotFoundError("recurring run", job.K8SName), "Failed to enable recurring run %v. Check if its k8s resource exists", jobId)
		}
	}

	_, err = r.getScheduledWorkflowClient(k8sNamespace).Patch(
		ctx,
		job.K8SName,
		types.MergePatchType,
		[]byte(fmt.Sprintf(`{"spec":{"enabled":%s}}`, strconv.FormatBool(enable))),
	)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to change recurring run's %v mode to enable:%v", jobId, enable)
	}

	err = r.jobStore.ChangeJobMode(jobId, enable)
	if err != nil {
		return util.Wrapf(err, "Failed to change recurring run's %v mode to enable:%v", jobId, enable)
	}
	return nil
}

// Deletes a recurring run with given id.
func (r *ResourceManager) DeleteJob(ctx context.Context, jobID string, propagationPolicy apiv2beta1.DeletePropagationPolicy) error {
	job, err := r.GetJob(jobID)
	if err != nil {
		return util.Wrapf(err, "Failed to delete recurring run %v. Check if exists", jobID)
	}

	k8sNamespace := job.Namespace
	if k8sNamespace == "" {
		k8sNamespace = common.GetPodNamespace()
	}

	deleteOptions := &v1.DeleteOptions{}
	if policy, exists := propagationPolicyMap[propagationPolicy]; exists {
		deleteOptions.PropagationPolicy = &policy
	}

	err = r.getScheduledWorkflowClient(k8sNamespace).Delete(ctx, job.K8SName, deleteOptions)
	if err != nil {
		if !util.IsNotFound(err) {
			return util.NewInternalServerError(err, "Failed to delete recurring run %v. Check if the scheduled workflow exists", jobID)
		}
		// The ScheduledWorkflow was not found.
		glog.Infof("Deleting recurring run '%v', but skipped deleting ScheduledWorkflow '%v' in namespace '%v' (k8s namespace %v) because it was not found", jobID, job.K8SName, job.Namespace, k8sNamespace)
		// Continue the execution, because we want to delete the
		// ScheduledWorkflow. We can skip deleting the ScheduledWorkflow
		// when it no longer exists.
	}
	err = r.jobStore.DeleteJob(jobID)
	if err != nil {
		return util.Wrapf(err, "Failed to delete recurring run %v", jobID)
	}
	return nil
}

// Creates new tasks or updates existing ones.
// This is not a part of internal API exposed to persistence agent only.
func (r *ResourceManager) CreateOrUpdateTasks(t []*model.Task, runID, workflowNamespace string) ([]*model.Task, error) {
	run, err := r.GetRun(runID)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to validate task ownership for run %s", runID)
	}
	return r.CreateOrUpdateTasksForRun(t, run, workflowNamespace)
}

// CreateOrUpdateTasksForRun creates or updates tasks using an already-loaded owning run.
func (r *ResourceManager) CreateOrUpdateTasksForRun(
	tasksToReport []*model.Task,
	run *model.Run,
	workflowNamespace string,
) ([]*model.Task, error) {
	if run == nil {
		return nil, util.NewInvalidInputError("Failed to report tasks: owning run is missing")
	}
	runID := run.UUID
	runNamespace, err := r.resolveWorkflowReportNamespace(
		"run", runID, run.Namespace, run.ExperimentId, workflowNamespace)
	if err != nil {
		r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
		return nil, err
	}
	if err := r.validateWorkflowReportNamespace(
		"run", runID, runNamespace, workflowNamespace, run.K8SName); err != nil {
		return nil, err
	}
	for _, task := range tasksToReport {
		if task.RunID != runID {
			r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
			return nil, util.NewInvalidInputError(
				"Failed to report tasks: task run ID does not match owning run")
		}
		if !r.IsEmptyNamespace(task.Namespace) && task.Namespace != workflowNamespace {
			r.recordWorkflowReportRejection(workflowReportRejectionNamespaceMismatch)
			return nil, util.NewInvalidInputError(
				"Failed to report tasks: task namespace does not match owning run")
		}
	}
	tasks, updated, err := r.taskStore.CreateOrUpdateTasksIfRunUnchanged(
		tasksToReport,
		runID,
		run.Namespace,
		run.WorkflowRuntimeManifest,
		run.PipelineRuntimeManifest,
		run.RetryGeneration,
	)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create or update tasks")
	}
	if !updated {
		// The workflow report and task upsert are separate transactions. If
		// the run changed between them, reload its authoritative identity and
		// retry the complete report rather than attaching stale tasks to a
		// replacement row that reused the run ID.
		r.storedWorkflowIdentities.delete(runID)
		currentRun, readError := r.GetRun(runID)
		if readError != nil {
			return nil, util.NewUnavailableServerError(
				readError,
				"Failed to reload run %s after a concurrent task report - try again later",
				runID,
			)
		}
		if _, identityError := r.storedWorkflowIdentityForRun(currentRun); identityError != nil {
			return nil, identityError
		}
		return nil, util.NewUnavailableServerError(
			errors.New("stored run changed while processing task report"),
			"Failed to report tasks for run %s because the stored run changed concurrently - try again later",
			runID,
		)
	}
	return tasks, nil
}

// Reports a workflow CR.
// This is called to update runs.
func (r *ResourceManager) ReportWorkflowResource(ctx context.Context, execSpec util.ExecutionSpec) (util.ExecutionSpec, error) {
	return r.reportWorkflowResource(ctx, execSpec, nil)
}

// ReportWorkflowResourceWithRun reports a workflow using an already-loaded owning run.
func (r *ResourceManager) ReportWorkflowResourceWithRun(
	ctx context.Context,
	execSpec util.ExecutionSpec,
	run *model.Run,
) (util.ExecutionSpec, error) {
	if run == nil {
		return nil, util.NewInvalidInputError("Failed to report workflow: owning run is missing")
	}
	return r.reportWorkflowResource(ctx, execSpec, run)
}

func (r *ResourceManager) reportWorkflowResource(
	ctx context.Context,
	execSpec util.ExecutionSpec,
	run *model.Run,
) (util.ExecutionSpec, error) {
	objMeta := execSpec.ExecutionObjectMeta()
	execStatus := execSpec.ExecutionStatus()
	if _, ok := objMeta.Labels[util.LabelKeyWorkflowRunId]; !ok {
		// Skip reporting if the workflow doesn't have the run id label
		return nil, util.NewInvalidInputError("Workflow[%s] missing the Run ID label", execSpec.ExecutionName())
	}
	runId := objMeta.Labels[util.LabelKeyWorkflowRunId]
	jobId := execSpec.ScheduledWorkflowUUIDAsStringOrEmpty()
	if len(execSpec.ExecutionNamespace()) == 0 {
		return nil, util.NewInvalidInputError("Failed to report a workflow. Namespace is empty")
	}
	// Evaluate the effective status at return time because identity validation
	// can replace a stale non-terminal snapshot with the terminal live workflow.
	defer func() {
		if execStatus.IsInFinalState() {
			r.storedWorkflowIdentities.delete(runId)
		}
	}()

	// If the run was Running and got terminated (activeDeadlineSeconds set to 0),
	// ignore its condition and mark it as such
	state := model.RuntimeState(string(execStatus.Condition())).ToV2()
	if execSpec.IsTerminating() {
		state = model.RuntimeState(string(exec.ExecutionPhase(model.RunTerminatingConditionsV1))).ToV2()
	}
	var verifiedLiveWorkflow util.ExecutionSpec
	if execStatus.IsInFinalState() {
		var workflowStillMatchesReport bool
		var err error
		verifiedLiveWorkflow, workflowStillMatchesReport, err = r.workflowStillMatchesReportedVersion(ctx, execSpec)
		if err != nil {
			return nil, err
		}
		if !workflowStillMatchesReport {
			return nil, terminalWorkflowReportDeferredError(
				runId,
				execSpec,
				"workflow resource version changed before terminal report was persisted",
			)
		}
	}
	// If run already exists, simply update it
	var updateError error
	if run == nil {
		run, updateError = r.GetRun(runId)
	} else if run.UUID != runId {
		return nil, util.NewInvalidInputError(
			"Failed to report workflow: provided run does not match the workflow run ID")
	}
	if updateError != nil && !util.IsUserErrorCodeMatch(updateError, codes.NotFound) {
		// Fail closed: a transient run-store read error must not skip the
		// generation fence below and fall through into the
		// persisted-final-state deletion - with an empty-resourceVersion
		// stale snapshot that would delete the live retried workflow.
		// NotFound keeps its dedicated recovery paths (workflow GC and the
		// grace-period handling further down).
		return nil, util.Wrapf(updateError, "Failed to read run %s before applying workflow report", runId)
	}
	var expectedWorkflowRuntimeManifest model.LargeText
	var expectedPipelineRuntimeManifest model.LargeText
	var verifiedExistingWorkflow util.ExecutionSpec
	existingWorkflowMissing := false
	var expectedStoredWorkflowIdentityManifest model.LargeText
	if updateError == nil {
		expectedWorkflowRuntimeManifest = run.WorkflowRuntimeManifest
		expectedPipelineRuntimeManifest = run.PipelineRuntimeManifest
		expectedStoredWorkflowIdentityManifest = storedWorkflowIdentityManifest(run)
		legacySingleUserRow := !common.IsMultiUserMode() && r.IsEmptyNamespace(run.Namespace)
		var legacyStoredIdentity storedWorkflowIdentity
		if legacySingleUserRow {
			var err error
			legacyStoredIdentity, err = r.storedWorkflowIdentityForRun(run)
			if err != nil {
				return nil, err
			}
		}
		modelNamespace := run.Namespace
		if legacySingleUserRow && legacyStoredIdentity.namespace != "" {
			modelNamespace = legacyStoredIdentity.namespace
		}
		runNamespace, err := r.resolveWorkflowReportNamespace(
			"run", runId, modelNamespace, run.ExperimentId, execSpec.ExecutionNamespace())
		if err != nil {
			r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
			return nil, err
		}
		if err := r.validateWorkflowReportNamespace("run", runId, runNamespace, execSpec.ExecutionNamespace(), execSpec.ExecutionName()); err != nil {
			return nil, err
		}
		if run.K8SName == "" {
			if legacyStoredIdentity.name != "" && legacyStoredIdentity.name != execSpec.ExecutionName() {
				return nil, r.validateWorkflowReportName(
					runId, legacyStoredIdentity.name, execSpec.ExecutionName())
			}
			liveWorkflow, err := r.validateLiveWorkflowReportIdentity(
				ctx, execSpec, verifiedLiveWorkflow, runId, jobId, "", false)
			if err != nil {
				if !util.IsUserErrorCodeMatch(err, codes.NotFound) || !execStatus.IsInFinalState() {
					r.recordWorkflowReportLiveLookupRejection(err)
					return nil, err
				}
				if err := r.validateStoredWorkflowReportIdentity(run, execSpec); err != nil {
					return nil, err
				}
				existingWorkflowMissing = true
				run.K8SName = execSpec.ExecutionName()
				if legacySingleUserRow {
					run.Namespace = execSpec.ExecutionNamespace()
				}
			} else {
				verifiedLiveWorkflow = liveWorkflow
				verifiedExistingWorkflow = liveWorkflow
				run.K8SName = liveWorkflow.ExecutionName()
			}
		} else if run.K8SName != execSpec.ExecutionName() {
			storedIdentity, err := r.storedWorkflowIdentityForRun(run)
			if err != nil {
				return nil, err
			}
			if storedIdentity.name != execSpec.ExecutionName() {
				return nil, r.validateWorkflowReportName(runId, run.K8SName, execSpec.ExecutionName())
			}
			// The immutable identity saved in the runtime manifest is
			// authoritative when a legacy Name column has diverged. The live
			// workflow and its UID are validated below before this correction is
			// persisted.
			run.K8SName = execSpec.ExecutionName()
		}
		if err := r.validateWorkflowReportRecurringRun(ctx, run, jobId, execSpec); err != nil {
			return nil, err
		}
		if verifiedExistingWorkflow == nil && !existingWorkflowMissing {
			recurringWorkflowName, err := r.recurringWorkflowNameForReport(jobId)
			if err != nil {
				return nil, err
			}
			verifiedExistingWorkflow, err = r.validateLiveWorkflowReportIdentity(
				ctx, execSpec, verifiedLiveWorkflow, runId, jobId, recurringWorkflowName, false)
			if err != nil {
				if !util.IsUserErrorCodeMatch(err, codes.NotFound) || !execStatus.IsInFinalState() {
					r.recordWorkflowReportLiveLookupRejection(err)
					return nil, err
				}
				if err := r.validateStoredWorkflowReportIdentity(run, execSpec); err != nil {
					return nil, err
				}
				existingWorkflowMissing = true
			}
			verifiedLiveWorkflow = verifiedExistingWorkflow
		}
		if verifiedExistingWorkflow != nil {
			// A report matching the current live object is not sufficient: an
			// editor can delete and recreate a same-name Workflow with copied
			// labels. Bind the live object to the immutable UID saved for this run
			// before allowing it to replace any persisted state. A pre-namespace
			// single-user row may adopt identity only when its stored manifest
			// genuinely lacks an immutable UID; otherwise its stored UID and any
			// stored namespace remain authoritative.
			legacyIdentityAdoption := legacySingleUserRow && legacyStoredIdentity.uid == ""
			if !legacyIdentityAdoption {
				_, err := r.validateStoredOrAdoptRetryWorkflowReportIdentity(run, verifiedExistingWorkflow)
				if err != nil {
					return nil, err
				}
			}
			execSpec = verifiedExistingWorkflow
			objMeta = execSpec.ExecutionObjectMeta()
			execStatus = execSpec.ExecutionStatus()
			state = workflowReportState(execSpec)
			if legacySingleUserRow {
				run.Namespace = execSpec.ExecutionNamespace()
			}
		}
	}

	// Resolve and validate a recurring run's owning namespace before any
	// workflow deletion. A missing run row is normal for the first report from
	// a ScheduledWorkflow, but the owner reference alone is not proof that the
	// reporting workflow belongs to that recurring run.
	var recurringJob *model.Job
	var recurringExperimentID string
	var recurringNamespace string
	if updateError != nil && util.IsUserErrorCodeMatch(updateError, codes.NotFound) && jobId != "" {
		var err error
		recurringJob, recurringExperimentID, recurringNamespace, err = r.resolveRecurringWorkflowReport(
			jobId, execSpec.ExecutionNamespace())
		if err != nil {
			r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
			glog.Errorf("Cannot establish ownership for workflow name=%q namespace=%q runId=%q recurringRunId=%q; refusing deletion and leaving the workflow for explicit cleanup: %v",
				execSpec.ExecutionName(), execSpec.ExecutionNamespace(), runId, jobId, err)
			return nil, util.Wrapf(err, "Failed to report a workflow for run %s due to error resolving recurring run %s", runId, jobId)
		}
		if err := r.validateWorkflowReportNamespace("recurring run", jobId, recurringNamespace, execSpec.ExecutionNamespace(), execSpec.ExecutionName()); err != nil {
			return nil, err
		}
		liveWorkflow, err := r.validateLiveWorkflowReportIdentity(
			ctx, execSpec, verifiedLiveWorkflow, runId, jobId, recurringJob.K8SName, false)
		if err != nil {
			r.recordWorkflowReportLiveLookupRejection(err)
			return nil, err
		}
		verifiedLiveWorkflow = liveWorkflow
		execSpec = liveWorkflow
		objMeta = execSpec.ExecutionObjectMeta()
		execStatus = execSpec.ExecutionStatus()
		state = workflowReportState(execSpec)
	}
	if updateError != nil && util.IsUserErrorCodeMatch(updateError, codes.NotFound) && jobId == "" {
		liveWorkflow, err := r.validateLiveWorkflowReportIdentity(
			ctx, execSpec, verifiedLiveWorkflow, runId, "", "", false)
		if err != nil {
			r.recordWorkflowReportLiveLookupRejection(err)
			return nil, err
		}
		execSpec = liveWorkflow
		objMeta = execSpec.ExecutionObjectMeta()
		// Preserve the startup grace period for an in-flight DB write. After it
		// expires, the live UID and labels establish which orphan is safe to
		// remove without trusting request metadata or deleting a replacement.
		gracePeriod := time.Duration(common.GetWorkflowGCGracePeriodSeconds()) * time.Second
		workflowAge := r.time.Now().Sub(objMeta.CreationTimestamp.Time)
		if workflowAge < gracePeriod {
			glog.Warningf(
				"Workflow name=%q namespace=%q runId=%q not found in run store, "+
					"but workflow is only %v old (grace period: %v). "+
					"Skipping report to allow an in-flight DB write to complete.",
				execSpec.ExecutionName(), execSpec.ExecutionNamespace(), runId,
				workflowAge.Round(time.Second), gracePeriod)
			return nil, util.NewUnavailableServerError(
				fmt.Errorf("workflow %s is within run-creation grace period (%v old, threshold %v)",
					execSpec.ExecutionName(), workflowAge.Round(time.Second), gracePeriod),
				"Skipping report for workflow %s - will retry",
				execSpec.ExecutionName())
		}
		deleteOperation := func() error {
			currentWorkflow, err := r.validateLiveWorkflowReportIdentity(
				ctx, execSpec, nil, runId, "", "", false)
			if util.IsUserErrorCodeMatch(err, codes.NotFound) {
				return nil
			}
			if err != nil {
				if util.IsUserErrorCodeMatch(err, codes.InvalidArgument) {
					return backoff.Permanent(err)
				}
				return err
			}
			if err := r.deleteLiveWorkflow(ctx, currentWorkflow); err != nil && !util.IsNotFound(err) {
				return err
			}
			return nil
		}
		if err := backoff.Retry(deleteOperation, newStandardBackoffPolicy()); err != nil {
			// backoff v2 unwraps PermanentError before returning, so preserve
			// the original client-visible classification here.
			if util.IsUserErrorCodeMatch(err, codes.InvalidArgument) {
				return nil, err
			}
			return nil, util.NewInternalServerError(
				err, "Failed to delete orphaned workflow for missing run %s after multiple retries", runId)
		}
		if r.options.CollectMetrics {
			workflowGCCounter.Inc()
		}
		return nil, util.Wrapf(updateError, "Deleted orphaned workflow for missing run %s", runId)
	}

	// Persist a newly observed recurring run before processing a pre-existing
	// persisted-final-state label. The label proves that this Workflow was
	// handled previously, but it does not prove that this API server's database
	// already contains the run row.
	createdFromRecurringReport := false
	if jobId != "" && updateError != nil {
		experimentID := recurringExperimentID
		namespace := recurringNamespace
		pipelineSpec := recurringJob.PipelineSpec
		pipelineSpec.WorkflowSpecManifest = model.LargeText(execSpec.GetExecutionSpec().ToStringForStore())
		scheduledTimeInSec := execSpec.ScheduledAtInSecOr0()
		if scheduledTimeInSec == 0 {
			scheduledTimeInSec = objMeta.CreationTimestamp.Unix()
		}
		proposedRun := &model.Run{
			UUID:           runId,
			ExperimentId:   experimentID,
			RecurringRunId: jobId,
			DisplayName:    execSpec.ExecutionName(),
			K8SName:        execSpec.ExecutionName(),
			StorageState:   model.StorageStateAvailable,
			Namespace:      namespace,
			PipelineSpec:   pipelineSpec,
			RunDetails: model.RunDetails{
				WorkflowRuntimeManifest: model.LargeText(execSpec.ToStringForStore()),
				CreatedAtInSec:          objMeta.CreationTimestamp.Unix(),
				ScheduledAtInSec:        scheduledTimeInSec,
				FinishedAtInSec:         execStatus.FinishedAt(),
				Conditions:              string(state.ToV1()),
				State:                   state,
			},
		}
		createdRun, err := r.runStore.CreateRun(proposedRun)
		if r.options.CollectMetrics && !execStatus.StartedAtTime().Time.IsZero() {
			reportGap := time.Since(execStatus.StartedAtTime().Time).Seconds()
			recurringPipelineRunReportGap.Observe(reportGap)
		}
		if err != nil {
			return nil, util.Wrapf(err, "Failed to report a workflow due to error creating run %s", runId)
		}
		if err := r.validateRecurringRunAfterCreate(
			createdRun,
			jobId,
			experimentID,
			namespace,
			execSpec,
		); err != nil {
			return nil, err
		}
		run = createdRun
		runId = run.UUID
		updateError = nil
		createdFromRecurringReport = true
		if err := r.experimentStore.SetLastRunTimestamp(run); err != nil {
			return nil, util.Wrapf(err, "Failed to report a workflow for existing run %s during updating the owning experiment.", runId)
		}
	}

	// Fence terminal reports from stale pre-retry workflow snapshots.
	// A retried workflow carries the claim's RetryGeneration as an
	// annotation; a snapshot taken before the retry carries an older
	// generation (or none). Accepting such a report would restore the
	// old FinishedAtInSec and make the run GC-eligible while the retry
	// is still running. No timestamps are compared across clocks here.
	// This fence runs before the persisted-final-state deletion below so a
	// stale snapshot can neither overwrite the row nor delete the live
	// workflow that a retry has since resubmitted under the same name.
	if updateError == nil && run.RetryGeneration > 0 && execStatus.IsInFinalState() {
		if reportedGeneration := reportedRetryGeneration(objMeta); reportedGeneration < run.RetryGeneration {
			claimAge := r.time.Now().Unix() - run.RetryClaimedAtInSec
			if run.RetryClaimedAtInSec > 0 && claimAge <= int64(retryClaimGracePeriod()/time.Second) {
				// The retry is (or was moments ago) in flight; the retried
				// workflow will report with the current generation. Skip
				// this stale snapshot as a successful no-op so the
				// persistence agent does not requeue it forever.
				glog.Infof("Skipping stale terminal report for run %s: reported retry generation %d < claimed generation %d",
					runId, reportedGeneration, run.RetryGeneration)
				return execSpec, nil
			}
			// The claim has aged out, but age alone is not proof of
			// abandonment (and the resource-version check above passes
			// vacuously for reports without a resourceVersion). Never
			// age-accept a lower generation while the claimed generation
			// is live: consult the live workflow first.
			liveWorkflow, readError := r.getWorkflowClient(execSpec.ExecutionNamespace()).Get(ctx, execSpec.ExecutionName(), v1.GetOptions{})
			switch {
			case readError == nil && liveWorkflow != nil &&
				reportedRetryGeneration(liveWorkflow.ExecutionObjectMeta()) >= run.RetryGeneration:
				glog.Infof("Skipping stale terminal report for run %s: live workflow carries generation %d",
					runId, reportedRetryGeneration(liveWorkflow.ExecutionObjectMeta()))
				return execSpec, nil
			case readError != nil && !util.IsNotFound(readError):
				return nil, util.NewUnavailableServerError(readError,
					"Cannot verify live workflow before accepting a stale-generation report for run %s - will retry", runId)
			}
			// Definitive: no live workflow, or the live workflow still
			// carries an older generation, so the claimed generation was
			// never applied. Accept this report so the run returns to its
			// last real state instead of staying PENDING forever.
			glog.Warningf("Accepting terminal report with stale retry generation %d for run %s: claim (generation %d) is older than %v and provably not applied",
				reportedGeneration, runId, run.RetryGeneration, retryClaimGracePeriod())
		}
	}

	var verifiedPersistedWorkflow util.ExecutionSpec
	if execSpec.PersistedFinalState() {
		if !execStatus.IsInFinalState() {
			r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
			return nil, util.NewInvalidInputError(
				"Failed to report workflow: persisted final state requires a terminal workflow")
		}
		if existingWorkflowMissing {
			verifiedPersistedWorkflow = execSpec
		} else {
			recurringWorkflowName, err := r.recurringWorkflowNameForReport(jobId)
			if err != nil {
				return nil, err
			}
			verifiedPersistedWorkflow, err = r.validateLiveWorkflowReportIdentity(
				ctx, execSpec, verifiedLiveWorkflow, runId, jobId, recurringWorkflowName, true)
			if err != nil {
				r.recordWorkflowReportLiveLookupRejection(err)
				return nil, err
			}
			execSpec = verifiedPersistedWorkflow
			execStatus = execSpec.ExecutionStatus()
			state = workflowReportState(execSpec)
		}
	}

	if updateError == nil && !createdFromRecurringReport {
		run.K8SName = execSpec.ExecutionName()
		run.State = state
		run.Conditions = string(state.ToV1())
		run.FinishedAtInSec = execStatus.FinishedAt()
		run.WorkflowRuntimeManifest = model.LargeText(execSpec.ToStringForStore())
		var updated bool
		updated, updateError = r.runStore.UpdateRunIfRuntimeManifestsUnchanged(
			run,
			expectedWorkflowRuntimeManifest,
			expectedPipelineRuntimeManifest,
		)
		if updateError != nil {
			return nil, util.Wrapf(updateError, "Failed to report a workflow for existing run %s during updating the run. Check if the run entry is corrupted", runId)
		}
		if !updated {
			// Another report changed the row after this request loaded it. Do
			// not let the stale snapshot delete a Workflow or persist tasks.
			// Refresh the process-local identity cache from the authoritative
			// row, then retry the complete RPC with a freshly loaded run.
			r.storedWorkflowIdentities.delete(runId)
			currentRun, readError := r.GetRun(runId)
			if readError != nil {
				return nil, util.NewUnavailableServerError(
					readError,
					"Failed to reload run %s after a concurrent workflow report - try again later",
					runId,
				)
			}
			if _, identityError := r.storedWorkflowIdentityForRun(currentRun); identityError != nil {
				return nil, identityError
			}
			return nil, util.NewUnavailableServerError(
				errors.New("stored run changed while processing workflow report"),
				"Failed to report workflow for run %s because the stored run changed concurrently - try again later",
				runId,
			)
		}
		r.storedWorkflowIdentities.replaceAfterPersist(
			runId,
			sha256.Sum256([]byte(expectedStoredWorkflowIdentityManifest)),
			storedWorkflowIdentity{
				name:            execSpec.ExecutionName(),
				namespace:       execSpec.ExecutionNamespace(),
				uid:             execSpec.ExecutionObjectMeta().UID,
				retryGeneration: run.RetryGeneration,
				manifestDigest:  sha256.Sum256([]byte(run.WorkflowRuntimeManifest)),
			})
	}
	// Delete a fully persisted workflow only after the version check above:
	// a stale snapshot carrying the persisted-final-state label must not
	// delete the live workflow object that a retry has since resubmitted
	// under the same name.
	if execSpec.PersistedFinalState() {
		// If workflow's final state has being persisted, the workflow should be garbage collected.
		err := r.deleteLiveWorkflow(ctx, verifiedPersistedWorkflow)
		if err != nil {
			if apierrors.IsConflict(err) {
				return nil, terminalWorkflowReportDeferredError(
					runId,
					execSpec,
					"workflow changed before persisted-final-state cleanup",
				)
			}
			// A fix for kubeflow/pipelines#4484, persistence agent might have an outdated item in its workqueue, so it will
			// report workflows that no longer exist. It's important to return a not found error, so that persistence
			// agent won't retry again.
			if util.IsNotFound(err) {
				return nil, util.NewNotFoundError(err, "Failed to delete the completed workflow for run %s", runId)
			} else {
				return nil, util.NewInternalServerError(err, "Failed to delete the completed workflow for run %s", runId)
			}
		}
		if r.options.CollectMetrics {
			workflowGCCounter.Inc()
		}
		// The run was finalized by an earlier report and the workflow has now
		// been deleted. Do not try to update or relabel the deleted object.
		execSpec.SetLabels(util.LabelKeyWorkflowRunId, runId)
		return execSpec, nil
	}
	if execStatus.IsInFinalState() {
		// Notify plugins of terminal state. If terminal handling cannot be
		// completed, return a retryable signal before callers report tasks or
		// workflow metrics from a stale terminal report.
		if run != nil && run.PluginsOutputString != nil && *run.PluginsOutputString != "" {
			pr, prErr := apiserverPlugins.ModelToPersistedRun(run, execSpec.ExecutionNamespace())
			if prErr != nil {
				glog.Warningf("Failed to build PersistedRun for plugin sync on run %q: %v", run.UUID, prErr)
			} else if !r.pluginDispatcher.OnRunEnd(ctx, pr) {
				glog.Warningf("Plugin sync failed for run %q; deferring persistedFinalState label so persistence agent retries", run.UUID)
				return nil, terminalWorkflowReportDeferredError(
					runId,
					execSpec,
					"plugin terminal sync requested retry",
				)
			}
		}

		stillMatchesReportedFinalState, err := r.runStillMatchesReportedFinalState(runId, state, execStatus.FinishedAt())
		if err != nil {
			return nil, err
		}
		if !stillMatchesReportedFinalState {
			return nil, terminalWorkflowReportDeferredError(
				runId,
				execSpec,
				"run state changed while reporting terminal workflow state",
			)
		}

		labelAdded, err := addWorkflowLabelIfWorkflowUnchanged(
			ctx,
			r.getWorkflowClient(execSpec.ExecutionNamespace()),
			execSpec.ExecutionName(),
			execSpec.Version(),
			util.LabelKeyWorkflowPersistedFinalState,
			"true",
		)
		if err != nil {
			message := fmt.Sprintf("Failed to add PersistedFinalState label to workflow %s", execSpec.ExecutionName())
			// A fix for kubeflow/pipelines#4484, persistence agent might have an outdated item in its workqueue, so it will
			// report workflows that no longer exist. It's important to return a not found error, so that persistence
			// agent won't retry again.
			if util.IsNotFound(err) {
				return nil, util.NewNotFoundError(err, "%s", message)
			} else {
				return nil, util.Wrapf(err, "%s", message)
			}
		}
		if !labelAdded {
			return nil, terminalWorkflowReportDeferredError(
				runId,
				execSpec,
				"workflow resource version changed before persistedFinalState label could be added",
			)
		}
		if r.options.CollectMetrics {
			execNamespace := execSpec.ExecutionNamespace()
			execName := execSpec.ExecutionName()

			if execStatus.Condition() == exec.ExecutionSucceeded {
				workflowSuccessCounter.WithLabelValues(execNamespace, execName).Inc()
			} else {
				errorMsg := execStatus.Message()
				// If workflow-level message is empty, try to get error from failed nodes
				if errorMsg == "" {
					if wf, ok := execSpec.(*util.Workflow); ok {
						for nodeID, node := range wf.Status.Nodes {
							if node.Phase == "Failed" || node.Phase == "Error" {
								if node.Message != "" {
									errorMsg = fmt.Sprintf("Node '%s' failed: %s", nodeID, node.Message)
									break
								}
							}
						}
					}
				}
				if errorMsg == "" {
					errorMsg = "(no error message available)"
				}
				glog.Errorf("pipeline '%s' finished with an error: %s", execName, errorMsg)

				// also collects counts regarding retries
				workflowFailedCounter.WithLabelValues(execNamespace, execName).Inc()
			}
		}
	}
	execSpec.SetLabels(util.LabelKeyWorkflowRunId, runId)
	return execSpec, nil
}

func (r *ResourceManager) resolveWorkflowReportNamespace(resourceType, resourceID, modelNamespace, experimentID, workflowNamespace string) (string, error) {
	if !r.IsEmptyNamespace(modelNamespace) {
		return modelNamespace, nil
	}
	if !common.IsMultiUserMode() {
		// Namespace isolation is disabled in single-user mode. Use the
		// workflow's actual namespace for legacy empty/model.NoNamespace rows
		// instead of today's API-server namespace: the latter may have changed
		// since the workflow was submitted and would permanently strand reports.
		return workflowNamespace, nil
	}
	if experimentID == "" {
		return "", util.NewInternalServerError(
			errors.New("owning experiment is missing"),
			"Failed to determine namespace for %s %s before applying workflow report", resourceType, resourceID,
		)
	}

	namespace, err := r.GetNamespaceFromExperimentId(experimentID)
	if err != nil {
		return "", util.NewInternalServerError(err,
			"Failed to determine namespace for %s %s before applying workflow report", resourceType, resourceID)
	}
	if r.IsEmptyNamespace(namespace) {
		return "", util.NewInternalServerError(
			errors.New("owning namespace is missing"),
			"Failed to determine namespace for %s %s before applying workflow report", resourceType, resourceID,
		)
	}
	return namespace, nil
}

func workflowReportState(execSpec util.ExecutionSpec) model.RuntimeState {
	state := model.RuntimeState(string(execSpec.ExecutionStatus().Condition())).ToV2()
	if execSpec.IsTerminating() {
		return model.RuntimeState(string(exec.ExecutionPhase(model.RunTerminatingConditionsV1))).ToV2()
	}
	return state
}

func (r *ResourceManager) recurringWorkflowNameForReport(jobID string) (string, error) {
	if jobID == "" {
		return "", nil
	}
	job, err := r.GetJob(jobID)
	if err != nil {
		if util.IsUserErrorCodeMatch(err, codes.NotFound) {
			// Orphan propagation can remove both the ScheduledWorkflow and the
			// Workflow owner reference. The immutable owner UID remains the
			// decisive check whenever the owner is still present.
			return "", nil
		}
		return "", util.Wrapf(err, "Failed to validate recurring run %s against its live workflow", jobID)
	}
	return job.K8SName, nil
}

func (r *ResourceManager) validateLiveWorkflowReportIdentity(
	ctx context.Context,
	reportedWorkflow util.ExecutionSpec,
	verifiedLiveWorkflow util.ExecutionSpec,
	runID,
	scheduledWorkflowID,
	scheduledWorkflowName string,
	requirePersistedFinalState bool,
) (util.ExecutionSpec, error) {
	liveWorkflow := verifiedLiveWorkflow
	if liveWorkflow == nil {
		var err error
		liveWorkflow, err = r.getWorkflowClient(reportedWorkflow.ExecutionNamespace()).Get(
			ctx, reportedWorkflow.ExecutionName(), v1.GetOptions{})
		if err != nil {
			if util.IsNotFound(err) {
				return nil, util.NewNotFoundError(
					err, "Failed to verify live workflow identity before reporting run %s", runID)
			}
			return nil, util.NewUnavailableServerError(
				err, "Failed to verify live workflow identity before reporting run %s - will retry", runID)
		}
	}

	reportedMeta := reportedWorkflow.ExecutionObjectMeta()
	liveMeta := liveWorkflow.ExecutionObjectMeta()
	liveScheduledWorkflowID := liveWorkflow.ScheduledWorkflowUUIDAsStringOrEmpty()
	identityMatches := reportedMeta.UID != "" &&
		liveMeta.UID != "" &&
		reportedMeta.UID == liveMeta.UID &&
		liveMeta.Labels[util.LabelKeyWorkflowRunId] == runID &&
		liveScheduledWorkflowID == scheduledWorkflowID
	if identityMatches && scheduledWorkflowID != "" && scheduledWorkflowName != "" {
		identityMatches = false
		for _, owner := range liveMeta.OwnerReferences {
			if string(owner.UID) == scheduledWorkflowID && owner.Name == scheduledWorkflowName {
				identityMatches = true
				break
			}
		}
	}
	if !identityMatches {
		r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
		glog.Warningf(
			"Rejecting workflow report whose live identity does not match: runID=%q namespace=%q workflowName=%q",
			runID, reportedWorkflow.ExecutionNamespace(), reportedWorkflow.ExecutionName())
		return nil, util.NewInvalidInputError(
			"Failed to report workflow: reported identity does not match the live workflow")
	}
	if requirePersistedFinalState &&
		(!liveWorkflow.PersistedFinalState() || !liveWorkflow.ExecutionStatus().IsInFinalState()) {
		r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
		return nil, util.NewInvalidInputError(
			"Failed to report workflow: live workflow is not in persisted final state")
	}
	return liveWorkflow, nil
}

func storedWorkflowIdentityManifest(run *model.Run) model.LargeText {
	storedManifest := run.WorkflowRuntimeManifest
	if storedManifest == "" {
		// V2 creation stores the authoritative created Argo Workflow in the
		// V2-facing runtime field. Before the first persistence-agent update,
		// WorkflowRuntimeManifest is intentionally empty, so use the equivalent
		// stored object to validate a terminal snapshot whose live CR is gone.
		storedManifest = run.PipelineRuntimeManifest
	}
	return storedManifest
}

func (r *ResourceManager) storedWorkflowIdentityForRun(run *model.Run) (storedWorkflowIdentity, error) {
	storedManifest := storedWorkflowIdentityManifest(run)
	if storedManifest == "" {
		r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
		return storedWorkflowIdentity{}, util.NewInternalServerError(
			errors.New("stored execution manifest is empty"),
			"Failed to verify stored workflow identity before reporting run %s", run.UUID)
	}
	// Cache the persisted workflow identity in a bounded LRU so node-transition
	// reports do not repeatedly decode a potentially large runtime manifest.
	// The manifest digest keeps the cache coherent with repairs persisted by
	// another API-server replica without requiring process-local invalidation.
	manifestDigest := sha256.Sum256([]byte(storedManifest))
	storedIdentity, found := r.storedWorkflowIdentities.load(run.UUID)
	if found && (storedIdentity.retryGeneration != run.RetryGeneration ||
		storedIdentity.manifestDigest != manifestDigest) {
		found = false
	}
	if !found {
		var storedWorkflow struct {
			Metadata v1.ObjectMeta `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(storedManifest), &storedWorkflow); err != nil {
			r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
			return storedWorkflowIdentity{}, util.NewInternalServerError(
				err, "Failed to verify stored workflow identity before reporting run %s", run.UUID)
		}
		storedIdentity = r.storedWorkflowIdentities.loadOrStore(run.UUID, storedWorkflowIdentity{
			name:            storedWorkflow.Metadata.Name,
			namespace:       storedWorkflow.Metadata.Namespace,
			uid:             storedWorkflow.Metadata.UID,
			retryGeneration: run.RetryGeneration,
			manifestDigest:  manifestDigest,
		})
	}
	return storedIdentity, nil
}

func storedWorkflowMatchesIdentity(storedIdentity storedWorkflowIdentity, reportedWorkflow util.ExecutionSpec) bool {
	reportedMeta := reportedWorkflow.ExecutionObjectMeta()
	return storedIdentity.uid != "" &&
		reportedMeta.UID != "" &&
		storedIdentity.uid == reportedMeta.UID &&
		storedIdentity.name == reportedWorkflow.ExecutionName() &&
		(storedIdentity.namespace == "" || storedIdentity.namespace == reportedWorkflow.ExecutionNamespace())
}

func retryWorkflowMatchesActiveClaim(
	run *model.Run,
	storedIdentity storedWorkflowIdentity,
	reportedWorkflow util.ExecutionSpec,
) bool {
	return run.State == model.RuntimeStatePending &&
		run.RetryGeneration > 0 &&
		storedIdentity.name == reportedWorkflow.ExecutionName() &&
		reportedRetryGeneration(reportedWorkflow.ExecutionObjectMeta()) == run.RetryGeneration
}

func (r *ResourceManager) validateStoredWorkflowReportIdentity(
	run *model.Run,
	reportedWorkflow util.ExecutionSpec,
) error {
	storedIdentity, err := r.storedWorkflowIdentityForRun(run)
	if err != nil {
		return err
	}
	if !storedWorkflowMatchesIdentity(storedIdentity, reportedWorkflow) {
		r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
		return util.NewInvalidInputError(
			"Failed to report workflow: reported identity does not match the stored workflow")
	}
	return nil
}

func (r *ResourceManager) validateStoredOrAdoptRetryWorkflowReportIdentity(
	run *model.Run,
	reportedWorkflow util.ExecutionSpec,
) (bool, error) {
	storedIdentity, err := r.storedWorkflowIdentityForRun(run)
	if err != nil {
		return false, err
	}
	if storedWorkflowMatchesIdentity(storedIdentity, reportedWorkflow) {
		return false, nil
	}
	if retryWorkflowMatchesActiveClaim(run, storedIdentity, reportedWorkflow) {
		return true, nil
	}
	r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
	return false, util.NewInvalidInputError(
		"Failed to report workflow: reported identity does not match the stored workflow")
}

func (r *ResourceManager) deleteLiveWorkflow(ctx context.Context, workflow util.ExecutionSpec) error {
	metadata := workflow.ExecutionObjectMeta()
	uid := metadata.UID
	if uid == "" {
		return util.NewInvalidInputError("Failed to delete workflow: live workflow UID is empty")
	}
	preconditions := &v1.Preconditions{UID: &uid}
	if metadata.ResourceVersion != "" {
		resourceVersion := metadata.ResourceVersion
		preconditions.ResourceVersion = &resourceVersion
	}
	return r.getWorkflowClient(workflow.ExecutionNamespace()).Delete(
		ctx,
		workflow.ExecutionName(),
		v1.DeleteOptions{Preconditions: preconditions},
	)
}

func (r *ResourceManager) resolveRecurringWorkflowReport(jobID, workflowNamespace string) (*model.Job, string, string, error) {
	job, err := r.GetJob(jobID)
	if err != nil {
		return nil, "", "", util.Wrapf(err, "Failed to retrieve recurring run %s", jobID)
	}
	experimentID := job.ExperimentId
	namespace := job.Namespace

	// Legacy job rows can rely on resource references rather than columns.
	if experimentID == "" {
		experimentRef, err := r.resourceReferenceStore.GetResourceReference(jobID, model.JobResourceType, model.ExperimentResourceType)
		if err != nil {
			// Only a missing job proves that a reported workflow is orphaned.
			// A job whose legacy ownership reference is absent is inconsistent
			// storage and must be retried/repaired rather than interpreted as
			// permission to garbage-collect a live workflow.
			return nil, "", "", util.NewInternalServerError(
				err,
				"Failed to retrieve the experiment ID for the job %v that created the run",
				jobID,
			)
		}
		experimentID = experimentRef.ReferenceUUID
		if r.IsEmptyNamespace(namespace) {
			if namespaceRef, err := r.resourceReferenceStore.GetResourceReference(jobID, model.JobResourceType, model.NamespaceResourceType); err == nil {
				namespace = namespaceRef.ReferenceUUID
			}
		}
	}
	if experimentID == "" {
		experimentID, err = r.GetDefaultExperimentId()
		if err != nil {
			return nil, "", "", util.NewInternalServerError(err, "Failed to fetch the default experiment for recurring run %s", jobID)
		}
	}
	namespace, err = r.resolveWorkflowReportNamespace(
		"recurring run", jobID, namespace, experimentID, workflowNamespace)
	if err != nil {
		return nil, "", "", err
	}
	return job, experimentID, namespace, nil
}

func (r *ResourceManager) validateWorkflowReportNamespace(resourceType, resourceID, expectedNamespace, workflowNamespace, executionName string) error {
	if r.IsEmptyNamespace(expectedNamespace) {
		r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
		return util.NewInvalidInputError(
			"Failed to report workflow: owning namespace cannot be determined",
		)
	}
	if expectedNamespace != workflowNamespace {
		r.recordWorkflowReportRejection(workflowReportRejectionNamespaceMismatch)
		glog.Warningf("Rejecting workflow namespace mismatch: resourceType=%q resourceID=%q expectedNamespace=%q reportedNamespace=%q executionName=%q",
			resourceType, resourceID, expectedNamespace, workflowNamespace, executionName)
		return util.NewInvalidInputError(
			"Failed to report workflow: reported namespace does not match owning resource")
	}
	return nil
}

func (r *ResourceManager) validateWorkflowReportName(runID, expectedName, reportedName string) error {
	if expectedName == "" || expectedName != reportedName {
		r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
		glog.Warningf(
			"Rejecting workflow name mismatch: runID=%q expectedName=%q reportedName=%q",
			runID, expectedName, reportedName)
		return util.NewInvalidInputError(
			"Failed to report workflow: reported name does not match owning run")
	}
	return nil
}

func (r *ResourceManager) validateWorkflowReportRecurringRun(
	ctx context.Context,
	run *model.Run,
	reportedRecurringRunID string,
	execSpec util.ExecutionSpec,
) error {
	runID := run.UUID
	expectedRecurringRunID := run.RecurringRunId
	if expectedRecurringRunID == reportedRecurringRunID {
		return nil
	}

	// Kubernetes removes a dependent Workflow's owner reference when its
	// ScheduledWorkflow is deleted with orphan propagation. Accept that one
	// asymmetric case only after the job row is gone and the report still
	// identifies the live Workflow object stored for this run.
	if expectedRecurringRunID != "" && reportedRecurringRunID == "" {
		_, jobErr := r.GetJob(expectedRecurringRunID)
		switch {
		case jobErr == nil:
			r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
			return util.NewInvalidInputError(
				"Failed to report workflow: recurring-run owner is missing while the recurring run still exists")
		case !util.IsUserErrorCodeMatch(jobErr, codes.NotFound):
			r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
			return util.Wrapf(jobErr,
				"Failed to verify orphaned workflow ownership for run %s", runID)
		}

		liveWorkflow, liveErr := r.getWorkflowClient(execSpec.ExecutionNamespace()).Get(
			ctx, execSpec.ExecutionName(), v1.GetOptions{})
		if liveErr != nil {
			if !util.IsNotFound(liveErr) {
				r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
				return util.NewUnavailableServerError(liveErr,
					"Cannot verify orphaned workflow ownership for run %s - will retry", runID)
			}
			// The persistence agent may have read the terminal orphan immediately
			// before TTL collection or manual deletion. The job row is already
			// gone, so accept only the exact immutable object stored for this run;
			// the caller's normal terminal-NotFound path will then persist the
			// snapshot before returning the final NotFound signal.
			if execSpec.ExecutionStatus().IsInFinalState() {
				if err := r.validateStoredWorkflowReportIdentity(run, execSpec); err != nil {
					return err
				}
				return nil
			}
			r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
			return util.NewInvalidInputError(
				"Failed to report workflow: orphaned workflow identity cannot be verified")
		}

		reportedMeta := execSpec.ExecutionObjectMeta()
		liveMeta := liveWorkflow.ExecutionObjectMeta()
		if reportedMeta.UID != "" &&
			liveMeta.UID != "" &&
			reportedMeta.UID == liveMeta.UID &&
			liveMeta.Labels[util.LabelKeyWorkflowRunId] == runID &&
			liveWorkflow.ScheduledWorkflowUUIDAsStringOrEmpty() == "" {
			return nil
		}
	}

	r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
	glog.Warningf(
		"Rejecting workflow recurring-run mismatch: runID=%q expectedRecurringRunID=%q reportedRecurringRunID=%q",
		runID, expectedRecurringRunID, reportedRecurringRunID)
	return util.NewInvalidInputError(
		"Failed to report workflow: reported owner does not match owning run")
}

func (r *ResourceManager) validateRecurringRunAfterCreate(
	run *model.Run,
	expectedRecurringRunID,
	expectedExperimentID,
	expectedNamespace string,
	workflow util.ExecutionSpec,
) error {
	if run == nil {
		return util.NewInternalServerError(
			errors.New("run store returned an empty run"),
			"Failed to validate recurring run after creation")
	}
	runNamespace, err := r.resolveWorkflowReportNamespace(
		"run", run.UUID, run.Namespace, run.ExperimentId, workflow.ExecutionNamespace())
	if err != nil {
		r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
		return err
	}
	if runNamespace != expectedNamespace ||
		run.RecurringRunId != expectedRecurringRunID ||
		run.ExperimentId != expectedExperimentID ||
		run.K8SName != workflow.ExecutionName() {
		r.recordWorkflowReportRejection(workflowReportRejectionIdentityMismatch)
		glog.Warningf(
			"Rejecting recurring workflow after run creation conflict: runID=%q recurringRunID=%q experimentID=%q namespace=%q workflowName=%q",
			run.UUID, run.RecurringRunId, run.ExperimentId, runNamespace, run.K8SName)
		return util.NewInvalidInputError(
			"Failed to report workflow: persisted run ownership does not match recurring run")
	}
	if err := r.validateWorkflowReportNamespace(
		"run", run.UUID, runNamespace, workflow.ExecutionNamespace(), workflow.ExecutionName()); err != nil {
		return err
	}
	// CreateRun returns an existing row when a concurrent report wins the
	// recurring-run insert. Mutable owner fields and the Workflow name can all
	// match across a same-name Kubernetes replacement, so bind the returned row
	// to the already-live-verified Workflow's immutable UID before accepting it.
	return r.validateStoredWorkflowReportIdentity(run, workflow)
}

func (r *ResourceManager) recordWorkflowReportRejection(reason string) {
	if r.options != nil && r.options.CollectMetrics {
		workflowReportRejectedCounter.WithLabelValues(reason).Inc()
	}
}

// recordWorkflowReportLiveLookupRejection counts only live-object lookup
// failures that make the current report permanently unusable. A transient
// Kubernetes read failure is retried, and a terminal NotFound may be accepted
// through stored identity validation, so callers invoke this only after
// deciding to return the lookup error.
func (r *ResourceManager) recordWorkflowReportLiveLookupRejection(err error) {
	if util.IsUserErrorCodeMatch(err, codes.NotFound) {
		r.recordWorkflowReportRejection(workflowReportRejectionOwnershipUnresolved)
	}
}

func terminalWorkflowReportDeferredError(runID string, execSpec util.ExecutionSpec, reason string) error {
	return util.NewUnavailableServerError(
		errors.New(reason),
		"Skipping terminal workflow report for run %s workflow %s: %s",
		runID,
		execSpec.ExecutionName(),
		reason,
	)
}

// retryClaimGracePeriod bounds how long a retry claim without a matching
// workflow write is trusted. RetryRun's claim-to-workflow-update window is
// bounded by a handful of client-side retries (seconds), so a claim this old
// with no workflow carrying its generation means the retry crashed mid-flight.
// Shared with ClaimRunForRetry's abandoned-claim takeover so both recovery
// paths age out together.
func retryClaimGracePeriod() time.Duration {
	return time.Duration(storage.RetryClaimGraceSeconds) * time.Second
}

// reportedRetryGeneration extracts the retry-generation annotation stamped by
// RetryRun. Workflows created before any retry (or before this annotation
// existed) return 0.
func reportedRetryGeneration(objMeta *v1.ObjectMeta) int64 {
	raw, ok := objMeta.Annotations[util.AnnotationKeyRetryGeneration]
	if !ok {
		return 0
	}
	generation, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		glog.Warningf("Ignoring malformed %s annotation %q", util.AnnotationKeyRetryGeneration, raw)
		return 0
	}
	return generation
}

func (r *ResourceManager) workflowStillMatchesReportedVersion(
	ctx context.Context,
	execSpec util.ExecutionSpec,
) (util.ExecutionSpec, bool, error) {
	reportedVersion := execSpec.Version()
	if reportedVersion == "" {
		return nil, true, nil
	}

	currentWorkflow, err := r.getWorkflowClient(execSpec.ExecutionNamespace()).Get(ctx, execSpec.ExecutionName(), v1.GetOptions{})
	if err != nil {
		if util.IsNotFound(err) {
			// The workflow CR is already gone, e.g. deleted between the
			// persistence agent's read and this report. Proceed with the
			// reported terminal state so the run row is still finalized;
			// the persistedFinalState label step then surfaces the NotFound
			// signal to the caller after the database write.
			glog.Warningf(
				"Workflow %q was not found while verifying the reported version; proceeding with the reported terminal state",
				execSpec.ExecutionName(),
			)
			return nil, true, nil
		}
		return nil, false, util.Wrapf(err, "Failed to verify current workflow version while reporting completed workflow %s", execSpec.ExecutionName())
	}
	if currentWorkflow.Version() == reportedVersion {
		return currentWorkflow, true, nil
	}

	glog.Warningf(
		"Skip reporting terminal workflow state for workflow %q because the workflow changed before terminal reporting: reported resourceVersion=%q, current resourceVersion=%q",
		execSpec.ExecutionName(),
		reportedVersion,
		currentWorkflow.Version(),
	)
	return currentWorkflow, false, nil
}

func (r *ResourceManager) runStillMatchesReportedFinalState(runID string, state model.RuntimeState, finishedAtInSec int64) (bool, error) {
	currentRun, err := r.GetRun(runID)
	if err != nil {
		return false, util.Wrapf(err, "Failed to verify current state for completed workflow report on run %s", runID)
	}
	if currentRun.State == state && currentRun.FinishedAtInSec == finishedAtInSec {
		return true, nil
	}

	glog.Warningf(
		"Skip adding persistedFinalState label for run %q because the run changed while reporting the terminal workflow state: reported state=%q finishedAt=%d, current state=%q finishedAt=%d",
		runID,
		state,
		finishedAtInSec,
		currentRun.State,
		currentRun.FinishedAtInSec,
	)
	return false, nil
}

type jsonPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// Adds a label only if the workflow still matches the object that reported the
// terminal state. This prevents a stale terminal report from labeling a retried
// workflow after RetryRun has updated the same workflow name back to Running.
func addWorkflowLabelIfWorkflowUnchanged(
	ctx context.Context,
	wfClient util.ExecutionInterface,
	name string,
	expectedResourceVersion string,
	labelKey string,
	labelValue string,
) (bool, error) {
	patchObj := []jsonPatchOperation{}
	if expectedResourceVersion != "" {
		patchObj = append(patchObj, jsonPatchOperation{
			Op:    "test",
			Path:  "/metadata/resourceVersion",
			Value: expectedResourceVersion,
		})
	}
	patchObj = append(patchObj, jsonPatchOperation{
		Op:    "add",
		Path:  "/metadata/labels/" + escapeJSONPointerPathPart(labelKey),
		Value: labelValue,
	})

	patch, err := json.Marshal(patchObj)
	if err != nil {
		return false, util.NewInternalServerError(err, "Unexpected error while marshaling a patch object")
	}

	operation := func() error {
		_, err = wfClient.Patch(ctx, name, types.JSONPatchType, patch, v1.PatchOptions{})
		if apierrors.IsConflict(err) {
			return backoff.Permanent(err)
		}
		return err
	}
	err = backoff.Retry(operation, newStandardBackoffPolicy())
	if permanentErr, ok := err.(*backoff.PermanentError); ok {
		err = permanentErr.Err
	}
	if apierrors.IsConflict(err) {
		glog.Warningf("Skip adding workflow label %q to workflow %q because the workflow changed while reporting the terminal state", labelKey, name)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func escapeJSONPointerPathPart(pathPart string) string {
	return strings.ReplaceAll(strings.ReplaceAll(pathPart, "~", "~0"), "/", "~1")
}

// Updates a recurring run with a scheduled workflow CR.
func (r *ResourceManager) ReportScheduledWorkflowResource(swf *util.ScheduledWorkflow) error {
	// Verify the job exists
	if _, err := r.GetJob(string(swf.UID)); err != nil {
		return util.Wrapf(err, "Failed to report scheduled workflow due to error retrieving recurring run %s", string(swf.UID))
	}
	return r.jobStore.UpdateJob(swf)
}

// Returns a workflow template based on the manifest in the following priority:
// 1. Pipeline spec manifest from an existing pipeline version,
// 2. Pipeline spec manifest or workflow spec manifest provided by a user.
// If an existing pipeline version is found, the referenced pipeline and pipeline version are updated.
func (r *ResourceManager) fetchTemplateFromPipelineSpec(pipelineSpec *model.PipelineSpec) (template.Template, string, error) {
	manifest := ""
	pipelineVersion, err := r.fetchPipelineVersionFromPipelineSpec(*pipelineSpec)
	if err != nil {
		return nil, "", util.Wrapf(err, "Failed to fetch a template due to error retrieving pipeline version")
	} else if pipelineVersion != nil {
		// Update references to the existing pipeline version
		pipelineSpec.PipelineId = pipelineVersion.PipelineId
		pipelineSpec.PipelineVersionId = pipelineVersion.UUID
		pipelineSpec.PipelineName = pipelineVersion.Name
		// Fetch the template from PipelineSpec field or the corresponding YAML file
		tempBytes, _, err := r.fetchTemplateFromPipelineVersion(pipelineVersion)
		if err != nil {
			return nil, "", util.Wrapf(err, "Failed to fetch a template due invalid manifest in pipeline version %v", pipelineSpec.PipelineVersionId)
		}
		manifest = string(tempBytes)
	} else {
		// Read the provided manifest and fail if it is empty
		manifest = string(pipelineSpec.PipelineSpecManifest)
		if manifest == "" {
			manifest = string(pipelineSpec.WorkflowSpecManifest)
		}
		if manifest == "" {
			return nil, "", util.NewInvalidInputError("Failed to fetch a template with an empty pipeline spec manifest")
		}
	}
	templateOptions := template.TemplateOptions{
		CacheDisabled:        r.options.CacheDisabled,
		DefaultWorkspace:     r.options.DefaultWorkspace,
		MLPipelineTLSEnabled: r.options.MLPipelineTLSEnabled,
		DefaultRunAsUser:     r.options.DefaultRunAsUser,
		DefaultRunAsGroup:    r.options.DefaultRunAsGroup,
		DefaultRunAsNonRoot:  r.options.DefaultRunAsNonRoot,
		DefaultHostUsers:     r.options.DefaultHostUsers,
	}
	tmpl, err := template.New([]byte(manifest), templateOptions)
	if err != nil {
		return nil, "", util.Wrap(err, "Failed to fetch a template with an invalid pipeline spec manifest")
	}
	return tmpl, manifest, nil
}

// Fetches PipelineSpec as []byte array and a new URI of PipelineSpec.
// Returns empty string if PipelineSpec is found via PipelineSpecURI.
// It attempts to fetch PipelineSpec in the following order:
//  1. Directly read from pipeline versions's PipelineSpec field.
//  2. Fetch a yaml file from object store based on pipeline versions's PipelineSpecURI field.
//  3. Fetch a yaml file from object store based on pipeline versions's id.
//  4. Fetch a yaml file from object store based on pipeline's id.
func (r *ResourceManager) fetchTemplateFromPipelineVersion(pipelineVersion *model.PipelineVersion) ([]byte, string, error) {
	if len(pipelineVersion.PipelineSpec) != 0 {
		// Check pipeline spec string first
		bytes := []byte(pipelineVersion.PipelineSpec)
		return bytes, string(pipelineVersion.PipelineSpecURI), nil
	} else {
		// Try reading object store from pipeline_spec_uri
		template, errURI := r.readPipelineSpecFromObjectStore(context.TODO(), string(pipelineVersion.PipelineSpecURI))
		if errURI != nil {
			// Try reading object store from pipeline_version_id
			template, errUUID := r.readPipelineSpecFromObjectStore(context.TODO(), r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.UUID)))
			if errUUID != nil {
				// Try reading object store from pipeline_id
				template, errPipelineID := r.readPipelineSpecFromObjectStore(context.TODO(), r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.PipelineId)))
				if errPipelineID != nil {
					return nil, "", util.Wrap(
						util.Wrap(
							util.Wrap(errURI, "Failed to read a file from pipeline_spec_uri"),
							util.Wrap(errUUID, "Failed to read a file from OS with pipeline_version_id").Error(),
						),
						util.Wrap(errPipelineID, "Failed to read a file from OS with pipeline_id").Error(),
					)
				}
				return template, r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.PipelineId)), nil
			}
			return template, r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.UUID)), nil
		}
		return template, "", nil
	}
}

func (r *ResourceManager) readPipelineSpecFromObjectStore(ctx context.Context, filePath string) ([]byte, error) {
	reader, err := r.objectStore.GetFileReader(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	limitedReader := io.LimitReader(reader, int64(common.MaxFileLength)+1)
	pipelineSpec, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to read pipeline spec from %v", filePath)
	}
	if len(pipelineSpec) > common.MaxFileLength {
		return nil, util.NewInvalidInputError(
			"Pipeline spec file size too large (%v bytes). Maximum supported size: %v.",
			len(pipelineSpec), common.MaxFileLength,
		)
	}
	return pipelineSpec, nil
}

// Creates the default experiment entry.
func (r *ResourceManager) CreateDefaultExperiment(namespace string) (string, error) {
	// First check that we don't already have a default experiment ID in the DB.
	defaultExperimentId, err := r.GetDefaultExperimentId()
	if err != nil {
		return "", util.Wrap(err, "Failed to check if default experiment exists")
	}
	// If default experiment ID is already present, don't fail, simply return.
	if defaultExperimentId != "" {
		glog.Infof("Default experiment already exists! ID: %v", defaultExperimentId)
		return defaultExperimentId, nil
	}

	// Check if an experiment named Default already exists
	defaultExperiment, err := r.experimentStore.GetExperimentByNameNamespace("Default", namespace)
	if err != nil || defaultExperiment == nil {
		// Create the default experiment
		defaultExperiment = &model.Experiment{
			Name:         "Default",
			Description:  "All runs created without specifying an experiment will be grouped here.",
			Namespace:    namespace,
			StorageState: model.StorageStateAvailable,
		}
		defaultExperiment, err = r.CreateExperiment(defaultExperiment)
		if err != nil {
			return "", util.Wrap(err, "Failed to create the default experiment")
		}
	}

	// Set default experiment ID in the DB
	err = r.SetDefaultExperimentId(defaultExperiment.UUID)
	if err != nil {
		return "", util.Wrap(err, "Failed to set default experiment ID")
	}

	glog.Infof("Default experiment is set. ID is: %v", defaultExperiment.UUID)
	return defaultExperiment.UUID, nil
}

// TODO(gkcalat): deprecate this as we no longer have metrics in the v2beta1 run message.
// Read metrics as ordinary artifacts instead.
// Creates a run metric entry.
func (r *ResourceManager) ReportMetric(metric *model.RunMetric) error {
	err := r.runStore.CreateMetric(metric)
	if err != nil {
		return util.Wrap(err, "Failed to report a run metric")
	}
	return nil
}

// ResolveArtifactPath resolves the object storage path for an artifact.
func (r *ResourceManager) ResolveArtifactPath(runID string, nodeID string, artifactName string) (string, error) {
	run, err := r.runStore.GetRun(runID)
	if err != nil {
		return "", err
	}
	if run.WorkflowRuntimeManifest == "" {
		return "", util.NewInvalidInputError("read artifact from run with v2 IR spec is not supported")
	}
	execSpec, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(run.WorkflowRuntimeManifest))
	if err != nil {
		return "", util.NewInternalServerError(
			err, "failed to unmarshal workflow '%s'", run.WorkflowRuntimeManifest)
	}
	artifactPath := execSpec.ExecutionStatus().FindObjectStoreArtifactKeyOrEmpty(nodeID, artifactName)
	if artifactPath == "" {
		return "", util.NewResourceNotFoundError(
			"artifact", common.CreateArtifactPath(runID, nodeID, artifactName))
	}
	return artifactPath, nil
}

// ReadArtifact streams artifact content from object storage to the provided writer.
func (r *ResourceManager) ReadArtifact(ctx context.Context, runID string, nodeID string, artifactName string, writer io.Writer) error {
	artifactPath, err := r.ResolveArtifactPath(runID, nodeID, artifactName)
	if err != nil {
		return err
	}

	reader, err := r.objectStore.GetFileReader(ctx, artifactPath)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to get file reader for %v", artifactPath)
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to stream artifact content")
	}

	return nil
}

// ObjectStore returns the object store interface for direct access to object storage operations
func (r *ResourceManager) ObjectStore() storage.ObjectStore {
	return r.objectStore
}

// Fetches the default experiment id.
func (r *ResourceManager) GetDefaultExperimentId() (string, error) {
	return r.defaultExperimentStore.GetDefaultExperimentId()
}

// Sets the default experiment id.
func (r *ResourceManager) SetDefaultExperimentId(id string) error {
	return r.defaultExperimentStore.SetDefaultExperimentId(id)
}

// Checks if sample pipelines have been loaded.
func (r *ResourceManager) HaveSamplesLoaded() (bool, error) {
	return r.dBStatusStore.HaveSamplesLoaded()
}

// Reports that sample pipelines have been loaded.
func (r *ResourceManager) MarkSampleLoaded() error {
	return r.dBStatusStore.MarkSampleLoaded()
}

// Creates a pipeline version.
// PipelineSpec is stored as a sting inside PipelineVersion in v2beta1.
func (r *ResourceManager) CreatePipelineVersion(pv *model.PipelineVersion) (*model.PipelineVersion, error) {
	// Extract pipeline id
	pipelineId := pv.PipelineId
	if len(pipelineId) == 0 {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due to missing pipeline id")
	}

	// Fetch pipeline spec
	pipelineSpecBytes, pipelineSpecURI, err := r.fetchTemplateFromPipelineVersion(pv)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version as template is broken")
	}
	pv.PipelineSpec = model.LargeText(string(pipelineSpecBytes))
	if pipelineSpecURI != "" {
		pv.PipelineSpecURI = model.LargeText(pipelineSpecURI)
	}

	// Create a template
	templateOptions := template.TemplateOptions{
		CacheDisabled:        r.options.CacheDisabled,
		DefaultWorkspace:     r.options.DefaultWorkspace,
		MLPipelineTLSEnabled: r.options.MLPipelineTLSEnabled,
		DefaultRunAsUser:     r.options.DefaultRunAsUser,
		DefaultRunAsGroup:    r.options.DefaultRunAsGroup,
		DefaultRunAsNonRoot:  r.options.DefaultRunAsNonRoot,
		DefaultHostUsers:     r.options.DefaultHostUsers,
	}
	tmpl, err := template.New(pipelineSpecBytes, templateOptions)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version due to template creation error")
	}
	if tmpl.GetTemplateType() == template.V1 {
		pipelineNamespace, _ := r.FetchNamespaceFromPipelineId(pipelineId)
		if pipelineNamespace == "" {
			pipelineNamespace = common.GetPodNamespace()
		}
		if util.IsV1PipelinesBlocked(pipelineNamespace) {
			return nil, util.NewInvalidInputError("V1 pipeline specs are not allowed. Please migrate to using KFP V2 pipelines.")
		}
	}
	// Validate pipeline's name in:
	// 1. pipeline spec for v2 pipelines and v2-compatible pipeline must comply with MLMD requirements
	// 2. display name must be non-empty
	pipelineSpecName := ""
	if tmpl.IsV2() {
		pipelineSpecName = tmpl.V2PipelineName()
		if err := common.ValidatePipelineName(pipelineSpecName); err != nil {
			return nil, err
		}
	}
	if pv.Name == "" {
		if pipelineSpecName == "" {
			return nil, util.NewInvalidInputError("pipeline version's name cannot be empty")
		}
		pv.Name = pipelineSpecName
	}

	if pv.DisplayName == "" {
		pv.DisplayName = pv.Name
	}

	// Parse parameters
	paramsJSON, err := tmpl.ParametersJSON()
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version due to error converting parameters to json")
	}
	pv.Parameters = model.LargeText(paramsJSON)
	pv.Status = model.PipelineVersionCreating
	pv.PipelineSpec = model.LargeText(string(tmpl.Bytes()))

	if err := model.ValidateTags(pv.Tags); err != nil {
		return nil, err
	}

	// Create a record in DB (tags are inserted in the same transaction to avoid deadlocks).
	version, err := r.pipelineStore.CreatePipelineVersion(pv)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create pipeline version in PipelineStore")
	}

	// After pipeline version being created in DB and pipeline file being
	// saved in minio server, set this pipeline version to status ready.
	version.Status = model.PipelineVersionReady
	err = r.pipelineStore.UpdatePipelineVersionStatus(version.UUID, version.Status)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to change the status of a new pipeline version with id %v", version.UUID)
	}
	return version, nil
}

// GetPipelineVersion returns a pipeline version by Id. Tags are loaded at the store level.
func (r *ResourceManager) GetPipelineVersion(pipelineVersionId string) (*model.PipelineVersion, error) {
	pipelineVersion, err := r.pipelineStore.GetPipelineVersion(pipelineVersionId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline version with id %v", pipelineVersionId)
	}
	return pipelineVersion, nil
}

// GetPipelineVersionByName returns a pipeline version by pipeline ID and version name. Tags are loaded at the store level.
func (r *ResourceManager) GetPipelineVersionByName(pipelineID, versionName string) (*model.PipelineVersion, error) {
	pipelineVersion, err := r.pipelineStore.GetPipelineVersionByName(pipelineID, versionName)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline version with pipelineID=%v and name=%v", pipelineID, versionName)
	}
	return pipelineVersion, nil
}

// GetLatestPipelineVersion returns the latest pipeline version for a specified pipeline id. Tags are loaded at the store level.
func (r *ResourceManager) GetLatestPipelineVersion(pipelineId string) (*model.PipelineVersion, error) {
	// Verify pipeline exists
	_, err := r.pipelineStore.GetPipeline(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get the latest pipeline version as pipeline was not found")
	}

	// Get the latest pipeline version
	latestPipelineVersion, err := r.pipelineStore.GetLatestPipelineVersion(pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get the latest pipeline version for a pipeline")
	}
	return latestPipelineVersion, nil
}

// ListPipelineVersions returns a list of pipeline versions. Tags are loaded at the store level.
// tagFilters is an optional map of tag key->value pairs to filter pipeline versions by.
func (r *ResourceManager) ListPipelineVersions(pipelineID string, opts *list.Options, tagFilters map[string]string) ([]*model.PipelineVersion, int, string, error) {
	pipelineVersions, totalSize, nextPageToken, err := r.pipelineStore.ListPipelineVersions(pipelineID, opts, tagFilters)
	if err != nil {
		err = util.Wrapf(err, "Failed to list pipeline versions with pipeline id %v, options %v", pipelineID, opts)
		return nil, 0, "", err
	}
	return pipelineVersions, totalSize, nextPageToken, nil
}

// Deletes a pipeline version and the corresponding PipelineSpec.
func (r *ResourceManager) DeletePipelineVersion(pipelineVersionId string) error {
	// Check if pipeline version exists
	_, err := r.pipelineStore.GetPipelineVersion(pipelineVersionId)
	if err != nil {
		return util.Wrapf(err, "Failed to delete pipeline version with id %v as it was not found", pipelineVersionId)
	}

	// Mark pipeline as deleting so it's not visible to user.
	err = r.pipelineStore.UpdatePipelineVersionStatus(pipelineVersionId, model.PipelineVersionDeleting)
	if err != nil {
		return util.Wrapf(err, "Failed to change the status of pipeline version id %v to DELETING", pipelineVersionId)
	}

	// Delete pipeline spec file and DB entry.
	// Not fail the request if this step failed. A background run will do the cleanup.
	// https://github.com/kubeflow/pipelines/issues/388
	// TODO(jingzhang36): For now (before exposing version API), we have only 1
	// file with both pipeline and version pointing to it;  so it is ok to do
	// the deletion as follows. After exposing version API, we can have multiple
	// versions and hence multiple files, and we shall improve performance by
	// either using async deletion in order for this method to be non-blocking
	// or or exploring other performance optimization tools provided by gcs.
	//

	// Delete the DB entry
	err = r.pipelineStore.DeletePipelineVersion(pipelineVersionId)
	if err != nil {
		glog.Errorf("%v", util.Wrapf(err, "Failed to delete a DB entry for pipeline version id %v", pipelineVersionId))
		return util.Wrapf(err, "Failed to delete a DB entry for pipeline version id %v", pipelineVersionId)
	}
	return nil
}

// Returns a template for a specified pipeline version id.
func (r *ResourceManager) GetPipelineVersionTemplate(pipelineVersionId string) ([]byte, error) {
	// Verify pipeline version exist
	pipelineVersion, err := r.pipelineStore.GetPipelineVersion(pipelineVersionId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get pipeline version template as pipeline version id %v was not found", pipelineVersionId)
	}

	// Fetch template []byte array
	if bytes, _, err := r.fetchTemplateFromPipelineVersion(pipelineVersion); err != nil {
		return nil, util.Wrapf(err, "Failed to get a template for pipeline version with id %v", pipelineVersionId)
	} else {
		return bytes, nil
	}
}

// Verifies whether the user identity, which is contained in the context object,
// can perform some action (verb) on a resource (resourceType/resourceName) living in the
// target namespace. If the returned error is nil, the authorization passes. Otherwise,
// authorization fails with a non-nil error.
func (r *ResourceManager) IsAuthorized(ctx context.Context, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authz if not multi-user mode.
		return nil
	}

	if common.IsMultiUserSharedReadMode() &&
		(resourceAttributes.Verb == common.RbacResourceVerbGet ||
			resourceAttributes.Verb == common.RbacResourceVerbList) {
		glog.Infof("Multi-user shared read mode is enabled. Request allowed: %+v", resourceAttributes)
		return nil
	}

	glog.Info("Getting user identity")
	if ctx == nil {
		return util.NewUnauthenticatedError(errors.New("Context is nil"), "Authentication request failed")
	}
	// If the request header contains the user identity, requests are authorized
	// based on the namespace field in the request.
	errlist := make([]error, 0)
	userIdentity := ""
	for _, auth := range r.authenticators {
		identity, err := auth.GetUserIdentity(ctx)
		if err == nil {
			userIdentity = identity

			break
		}
		errlist = append(errlist, err)
	}
	if userIdentity == "" {
		return util.NewUnauthenticatedError(utilerrors.NewAggregate(errlist), "Failed to check authorization. User identity is empty in the request header")
	}

	glog.Infof("User: %s, ResourceAttributes: %+v", userIdentity, resourceAttributes)
	glog.Info("Authorizing request")
	result, err := r.subjectAccessReviewClient.Create(
		ctx,
		&authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				ResourceAttributes: resourceAttributes,
				User:               userIdentity,
			},
		},
		v1.CreateOptions{},
	)
	if err != nil {
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			reportErr := util.NewUnavailableServerError(
				err,
				"Failed to create SubjectAccessReview for user '%s' (request: %+v) - try again later",
				userIdentity,
				resourceAttributes,
			)
			glog.Info(reportErr.Error())
			return reportErr
		} else {
			reportErr := util.NewInternalServerError(
				err,
				"Failed to create SubjectAccessReview for user '%s' (request: %+v)",
				userIdentity,
				resourceAttributes,
			)
			glog.Info(reportErr.Error())
			return reportErr
		}
	}
	if !result.Status.Allowed {
		err := util.NewPermissionDeniedError(
			errors.New("Unauthorized access"),
			"User '%s' is not authorized with reason: %s (request: %+v)",
			userIdentity,
			result.Status.Reason,
			resourceAttributes,
		)
		glog.Info(err.Error())
		return err
	}
	glog.Infof("Authorized user '%s': %+v", userIdentity, resourceAttributes)
	return nil
}

// Fetches namespace that an experiment belongs to.
func (r *ResourceManager) GetNamespaceFromExperimentId(experimentId string) (string, error) {
	if experimentId == "" {
		return "", nil
	}
	experiment, err := r.GetExperiment(experimentId)
	if err != nil {
		return "", util.Wrapf(err, "Failed to fetch namespace from experiment %v", experimentId)
	}
	if experiment.Namespace == "" {
		if common.IsMultiUserMode() {
			namespaceRef, err := r.resourceReferenceStore.GetResourceReference(experimentId, model.ExperimentResourceType, model.NamespaceResourceType)
			if err != nil {
				return "", util.Wrapf(err, "Failed to fetch namespace from experiment %v due to resource references fetching error", experimentId)
			}
			if namespaceRef == nil || namespaceRef.ReferenceUUID == "" {
				return "", util.NewInternalServerError(util.NewNotFoundError(errors.New("Namespace is empty"), "Experiment's namespace was not found"), "Failed to fetch a namespace for experiment %v in multi-user mode", experimentId)
			}
			experiment.Namespace = namespaceRef.ReferenceUUID
		} else {
			experiment.Namespace = ""
		}
	}
	return experiment.Namespace, nil
}

// Fetches namespace that a run belongs to.
func (r *ResourceManager) getNamespaceFromRunId(runId string) (string, error) {
	run, err := r.GetRun(runId)
	if err != nil {
		return "", util.Wrapf(err, "Failed to fetch namespace from run %v due to fetching error", runId)
	}
	if !r.IsEmptyNamespace(run.Namespace) {
		return run.Namespace, nil
	}
	namespace, err := r.GetNamespaceFromExperimentId(run.ExperimentId)
	if err != nil {
		return "", util.Wrapf(err, "Failed to fetch namespace from run %v", runId)
	}
	return namespace, nil
}

// Returns parent namespace for a pipeline id.
func (r *ResourceManager) FetchNamespaceFromPipelineId(pipelineId string) (string, error) {
	pipeline, err := r.GetPipeline(pipelineId)
	if err != nil {
		return "", util.Wrapf(err, "Failed to get namespace for pipeline id %v", pipelineId)
	}
	return pipeline.Namespace, nil
}

// Returns parent namespace for a pipeline version id.
func (r *ResourceManager) FetchNamespaceFromPipelineVersionId(versionId string) (string, error) {
	pipelineVersion, err := r.GetPipelineVersion(versionId)
	if err != nil {
		return "", util.Wrapf(err, "Failed to get namespace for pipeline version id %v", versionId)
	}
	return r.FetchNamespaceFromPipelineId(pipelineVersion.PipelineId)
}

// Checks if the namespace is empty or equal to `-`.
func (r *ResourceManager) IsEmptyNamespace(namespace string) bool {
	if namespace == "" || namespace == model.NoNamespace {
		return true
	}
	return false
}

// Replaces the namespace to a default value for single-user mode.
func (r *ResourceManager) ReplaceNamespace(namespace string) string {
	if common.IsMultiUserMode() {
		return namespace
	} else {
		return ""
	}
}

// Validates that the provided experiment belongs to the namespace. Returns error otherwise.
// Returns an error in multi-user mode when experimentId and namespace are both empty.
func (r *ResourceManager) CheckExperimentBelongsToNamespace(experimentId string, namespace string) error {
	if experimentId == "" || r.IsEmptyNamespace(namespace) {
		if common.IsMultiUserMode() {
			return util.NewInvalidInputError("Resource cannot have an empty namespace and experiment id in multi-user mode")
		}
		return nil
	}
	experimentNamespace, err := r.GetNamespaceFromExperimentId(experimentId)
	if err != nil {
		return util.Wrapf(err, "Failed to validate the namespace of experiment %s", experimentId)
	}
	if experimentNamespace != "" && experimentNamespace != namespace {
		return util.NewInvalidInputError("Failed to validate the namespace of experiment: experiment %s belongs to namespace '%s' (claimed a different namespace '%s')",
			experimentId, experimentNamespace, namespace)
	}
	return nil
}

// Validates the provided experimentId and namespace. Returns valid values if the provided ones are empty.
// For multi-user more at least one of the input must be non-empty, otherwise, returns an error.
//  1. Validates that given experimentId belongs to namespace if both are not empty
//  2. If experimentId is empty, replaces it with the default experimentId from the given namespace.
//     Creates the default experiment in the given namespace (could be empty in single-user mode) if it is missing.
//  3. Replaces empty namespace with the parent namespace of the given experimentId.
func (r *ResourceManager) GetValidExperimentNamespacePair(experimentId string, namespace string) (string, string, error) {
	if common.IsMultiUserMode() && experimentId == "" {
		return "", "", util.NewInvalidInputError("Experiment id can not be empty in multi-user mode")
	}
	if experimentId != "" {
		ns, err := r.GetNamespaceFromExperimentId(experimentId)
		if err != nil {
			return "", "", util.Wrapf(err, "Failed to fetch namespace for experiment %v", experimentId)
		}
		if namespace != "" && namespace != ns {
			return "", "", util.NewInvalidInputError("Experiment %v belongs to namespace '%v' instead of '%v'", experimentId, ns, namespace)
		}
		namespace = ns
	} else {
		defExpId, err := r.GetDefaultExperimentId()
		if err != nil {
			return "", "", util.Wrapf(err, "Specify experiment id or check if the default experiment exists in namespace %v", namespace)
		}
		// Create the default experiment if it is missing
		if defExpId == "" {
			defExpId, err = r.CreateDefaultExperiment(namespace)
			if err != nil {
				return "", "", util.Wrapf(err, "Experiment id is empty. Failed to create a new default experiment in namespace %v", namespace)
			}
		}
		experimentId = defExpId
	}
	return experimentId, namespace, nil
}

// Fetches a task entry.
func (r *ResourceManager) GetTask(taskId string) (*model.Task, error) {
	task, err := r.taskStore.GetTask(taskId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to fetch task %v", taskId)
	}
	return task, nil
}

func (r *ResourceManager) authorizeServiceAccount(ctx context.Context, serviceAccount, namespace string) error {
	if serviceAccount == "" {
		return nil
	}
	if err := common.ValidateServiceAccountAllowList(serviceAccount); err != nil {
		return util.NewInvalidInputError("%s", err)
	}
	defaultServiceAccount := common.GetStringConfigWithDefault(common.DefaultPipelineRunnerServiceAccountFlag, common.DefaultPipelineRunnerServiceAccount)
	if serviceAccount == defaultServiceAccount {
		return nil
	}
	return r.IsAuthorized(ctx, &authorizationv1.ResourceAttributes{
		Verb:      common.RbacResourceVerbUse,
		Namespace: namespace,
		Resource:  "serviceaccounts",
		Name:      serviceAccount,
	})
}
