package storage

import (
	"context"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"slices"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrNoV1             = errors.New("the v1 API is not available through Kubernetes")
	ErrUnsupportedField = errors.New("the field is unsupported")
)

type PipelineStoreKubernetes struct {
	client ctrlclient.Client
}

func NewPipelineStoreKubernetes(k8sClient ctrlclient.Client) *PipelineStoreKubernetes {
	return &PipelineStoreKubernetes{client: k8sClient}
}

func (k *PipelineStoreKubernetes) GetPipelineByNameAndNamespaceV1(name string, namespace string) (*model.Pipeline, *model.PipelineVersion, error) {
	return nil, nil, ErrNoV1
}

func (k *PipelineStoreKubernetes) GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error) {
	k8sPipeline := v2beta1.Pipeline{}

	err := k.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &k8sPipeline)
	if k8serrors.IsNotFound(err) {
		return nil, util.NewResourceNotFoundError("Namespace/Pipeline", fmt.Sprintf("%v/%v", namespace, name))
	} else if err != nil {
		return nil, util.NewInternalServerError(
			err, "Failed to get a pipeline with name %v and namespace %v", name, namespace,
		)
	}

	return k8sPipeline.ToModel(), nil
}

func (k *PipelineStoreKubernetes) ListPipelinesV1(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, []*model.PipelineVersion, int, string, error) {
	return nil, nil, 0, "", ErrNoV1
}

func (k *PipelineStoreKubernetes) ListPipelines(filterContext *model.FilterContext, opts *list.Options) ([]*model.Pipeline, int, string, error) {
	k8sPipelines := v2beta1.PipelineList{}

	listOptions := []ctrlclient.ListOption{ctrlclient.UnsafeDisableDeepCopy}

	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
		listOptions = append(listOptions, ctrlclient.InNamespace(filterContext.ReferenceKey.ID))
	}

	// Be careful, the deep copy is disabled here to reduce memory allocations
	err := k.client.List(context.TODO(), &k8sPipelines, listOptions...)
	if err != nil {
		return nil, 0, "", util.NewInternalServerError(
			err, "Failed to find the pipeline associated with this pipeline version",
		)
	}

	pipelines := make([]*model.Pipeline, 0, len(k8sPipelines.Items))

	for _, k8sPipeline := range k8sPipelines.Items {
		if opts.Filter == nil {
			pipelines = append(pipelines, k8sPipeline.ToModel())
			continue
		} else {
			found, err1 := opts.Filter.FilterK8sPipelines(k8sPipeline)
			if err1 != nil {
				return nil, 0, "", err1
			}
			if found {
				pipelines = append(pipelines, k8sPipeline.ToModel())
			}
		}
	}

	// Because controller-client does not have sorting, use this function to sort by fields.
	sort.Slice(pipelines, func(i, j int) bool {
		if strings.Compare(opts.SortByFieldName, "CreatedAtInSec") == 0 {
			return pipelines[i].CreatedAtInSec > pipelines[j].CreatedAtInSec
		} else if strings.Compare(opts.SortByFieldName, "Name") == 0 {
			return pipelines[i].Name > pipelines[j].Name
		}
		return false
	})
	if !opts.IsDesc {
		slices.Reverse(pipelines)
	}

	// If there's pagination, find the index of the first element of the next page
	nextPage := 0
	for p, pipeline := range pipelines {
		if strings.Compare(fmt.Sprint(opts.KeyFieldValue), pipeline.UUID) == 0 {
			nextPage = p

			break
		}
	}

	// Split results on multiple pages if needed
	upperBound := nextPage + opts.PageSize
	npt := ""
	if len(pipelines) <= opts.PageSize {
		return pipelines, len(pipelines), "", nil
	} else if upperBound >= len(pipelines) {
		upperBound = len(pipelines)
		npt, err = opts.NextPageToken(pipelines[upperBound-1])
	} else {
		npt, err = opts.NextPageToken(pipelines[upperBound])
	}
	return pipelines[nextPage:upperBound], opts.PageSize, npt, err
}

func (k *PipelineStoreKubernetes) GetPipeline(pipelineId string) (*model.Pipeline, error) {
	k8sPipeline, err := k.getK8sPipeline(pipelineId)
	if err != nil {
		return nil, err
	}

	return k8sPipeline.ToModel(), nil
}

func (k *PipelineStoreKubernetes) GetPipelineWithStatus(pipelineId string, status model.PipelineStatus) (*model.Pipeline, error) {
	pipeline, err := k.GetPipeline(pipelineId)
	if err != nil {
		return nil, err
	}

	if pipeline.Status != status {
		return nil, util.NewResourceNotFoundError("Pipeline", pipelineId)
	}

	return pipeline, nil
}

func (k *PipelineStoreKubernetes) DeletePipeline(pipelineId string) error {
	k8sPipeline, err := k.getK8sPipeline(pipelineId)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundError:") {
			return nil
		}
	}

	err = k.client.Delete(context.TODO(), k8sPipeline)
	if err != nil && !k8serrors.IsNotFound(err) {
		return util.NewInternalServerError(err, "Failed to delete the pipeline")
	}

	return nil
}

func (k *PipelineStoreKubernetes) CreatePipelineAndPipelineVersion(pipeline *model.Pipeline, pipelineVersion *model.PipelineVersion) (*model.Pipeline, *model.PipelineVersion, error) {
	pipeline.UUID = ""
	pipelineVersion.UUID = ""

	var err error

	pipeline, err = k.CreatePipeline(pipeline)
	if err != nil {
		return nil, nil, err
	}

	pipelineVersion, err = k.createPipelineVersionWithPipeline(context.TODO(), pipeline, pipelineVersion)
	if err != nil {
		return nil, nil, err
	}

	return pipeline, pipelineVersion, nil
}

func (k *PipelineStoreKubernetes) CreatePipeline(pipeline *model.Pipeline) (*model.Pipeline, error) {
	pipeline.UUID = ""

	if pipeline.Parameters != "" {
		return nil, util.NewBadRequestError(ErrUnsupportedField, "The parameters field is not supported")
	}

	if pipeline.Namespace == "" {
		if common.IsMultiUserMode() || common.GetPodNamespace() == "" {
			return nil, util.NewBadRequestError(errors.New("A namespace is required"), "")
		}

		pipeline.Namespace = common.GetPodNamespace()
	}

	pipeline.Status = model.PipelineCreating

	k8sPipeline := v2beta1.FromPipelineModel(*pipeline)

	glog.Infof("Creating the pipeline %s/%s in Kubernetes", k8sPipeline.Namespace, k8sPipeline.Name)

	err := k.client.Create(context.TODO(), &k8sPipeline)
	if k8serrors.IsAlreadyExists(err) {
		return nil, util.NewAlreadyExistError(
			"Failed to create a new pipeline. The name %v already exists. Please specify a new name", pipeline.Name,
		)
	} else if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create the pipeline")
	}

	return k8sPipeline.ToModel(), nil
}

func (k *PipelineStoreKubernetes) UpdatePipelineStatus(pipelineId string, status model.PipelineStatus) error {
	// Do nothing. Just return nil to avoid show an unrelated error
	return nil
}

func (k *PipelineStoreKubernetes) UpdatePipelineVersionStatus(pipelineVersionId string, status model.PipelineVersionStatus) error {
	k8sPipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionId)
	if err != nil {
		return err
	}

	conditionSet := false

	for i := range k8sPipelineVersion.Status.Conditions {
		condition := &k8sPipelineVersion.Status.Conditions[i]

		if condition.Type == "PipelineVersionStatus" {
			if condition.Reason == string(status) && condition.Message == condition.Reason {
				return nil
			}

			condition.Reason = string(status)
			condition.Message = string(status)
			condition.Status = metav1.ConditionTrue

			conditionSet = true

			break
		}
	}

	if !conditionSet {
		k8sPipelineVersion.Status.Conditions = append(k8sPipelineVersion.Status.Conditions, v2beta1.SimplifiedCondition{
			Type:    "PipelineVersionStatus",
			Reason:  string(status),
			Message: string(status),
			Status:  metav1.ConditionTrue,
		})
	}

	err = k.client.Status().Update(context.TODO(), k8sPipelineVersion)
	if err != nil && k8serrors.IsConflict(err) {
		return k.UpdatePipelineVersionStatus(pipelineVersionId, status)
	} else if err != nil {
		return util.NewInternalServerError(err, "Failed to update the pipeline version status")
	}
	return nil
}

func (k *PipelineStoreKubernetes) CreatePipelineVersion(pipelineVersion *model.PipelineVersion) (*model.PipelineVersion, error) {
	pipeline, err := k.GetPipeline(pipelineVersion.PipelineId)
	if err != nil {
		return nil, err
	}

	return k.createPipelineVersionWithPipeline(context.TODO(), pipeline, pipelineVersion)
}

func (k *PipelineStoreKubernetes) UpdatePipelineDefaultVersion(pipelineId string, versionId string) error {
	// Default version was used in KFPv1 and is deprecated. In KFPv2, we do not support this.
	return util.NewBadRequestError(errors.New("pipeline default version is unsupported"),
		"pipeline default version is unsupported when storing in Kubernetes")
}

func (k *PipelineStoreKubernetes) GetLatestPipelineVersion(pipelineId string) (*model.PipelineVersion, error) {
	k8sPipelineVersions, err := k.getK8sPipelineVersions(context.TODO(), pipelineId)
	if err != nil {
		return nil, err
	}

	var latestK8sPipelineVersion *v2beta1.PipelineVersion

	for _, k8sPipelineVersion := range k8sPipelineVersions.Items {
		if latestK8sPipelineVersion == nil || k8sPipelineVersion.CreationTimestamp.Time.After(latestK8sPipelineVersion.CreationTimestamp.Time) {
			latestK8sPipelineVersion = &k8sPipelineVersion

			continue
		}
	}

	if latestK8sPipelineVersion == nil {
		return nil, util.NewResourceNotFoundError("PipelineVersion", "Latest")
	}

	return latestK8sPipelineVersion.ToModel()
}

func (k *PipelineStoreKubernetes) GetPipelineVersion(pipelineVersionId string) (*model.PipelineVersion, error) {
	pipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionId)
	if err != nil {
		return nil, err
	}

	return pipelineVersion.ToModel()
}

func (k *PipelineStoreKubernetes) GetPipelineVersionByName(name string) (*model.PipelineVersion, error) {
	k8sPipelineVersions, err := k.getK8sPipelineVersions(context.TODO(), "")
	if err != nil {
		return nil, err
	}

	for _, k8sPipelineVersion := range k8sPipelineVersions.Items {
		if k8sPipelineVersion.Name == name {
			pipelineVersion, err1 := k8sPipelineVersion.ToModel()
			if err1 != nil {
				return nil, err1
			}

			return pipelineVersion, nil
		}
	}
	return nil, nil
}

func (k *PipelineStoreKubernetes) GetPipelineVersionWithStatus(pipelineVersionId string, status model.PipelineVersionStatus) (*model.PipelineVersion, error) {
	pipelineVersion, err := k.GetPipelineVersion(pipelineVersionId)
	if err != nil {
		return nil, err
	}

	if pipelineVersion.Status != status {
		return nil, util.NewResourceNotFoundError("PipelineVersion", pipelineVersionId)
	}

	return pipelineVersion, nil
}

func (k *PipelineStoreKubernetes) ListPipelineVersions(pipelineId string, opts *list.Options) (versions []*model.PipelineVersion, totalSize int, nextPageToken string, err error) {
	k8sPipelineVersions, err := k.getK8sPipelineVersions(context.TODO(), pipelineId)
	if err != nil {
		return nil, 0, "", err
	}

	pipelineVersions := make([]*model.PipelineVersion, 0, len(k8sPipelineVersions.Items))

	for _, k8sPipelineVersion := range k8sPipelineVersions.Items {
		pipelineVersion, err := k8sPipelineVersion.ToModel()
		if err != nil {
			return nil, 0, "", err
		}

		if opts.Filter == nil {
			pipelineVersions = append(pipelineVersions, pipelineVersion)
		} else {
			found, err1 := opts.Filter.FilterK8sPipelineVersions(k8sPipelineVersion)
			if err1 != nil {
				return nil, 0, "", err1
			} else if found {
				pipelineVersions = append(pipelineVersions, pipelineVersion)
			}
		}
	}

	// Because controller-client does not have sorting, use this function to sort by fields.
	sort.Slice(pipelineVersions, func(i, j int) bool {
		if elementA, ok1 := pipelineVersions[i].GetField(opts.SortByFieldName); ok1 {
			if elementB, ok2 := pipelineVersions[j].GetField(opts.SortByFieldName); ok2 {
				return elementA > elementB
			}
		}
		return false
	})
	if !opts.IsDesc {
		slices.Reverse(pipelineVersions)
	}

	// If there's pagination, find the index of the first element of the next page
	nextPage := 0
	for p, pipelineVersion := range pipelineVersions {
		if strings.Compare(fmt.Sprint(opts.KeyFieldValue), pipelineVersion.UUID) == 0 {
			nextPage = p

			break
		}
	}

	// Split results on multiple pages if needed
	upperBound := nextPage + opts.PageSize
	npt := ""
	if len(pipelineVersions) <= opts.PageSize {
		return pipelineVersions, len(pipelineVersions), "", nil
	} else if upperBound >= len(pipelineVersions) {
		upperBound = len(pipelineVersions)
		npt, err = opts.NextPageToken(pipelineVersions[upperBound-1])
	} else {
		npt, err = opts.NextPageToken(pipelineVersions[upperBound])
	}
	return pipelineVersions[nextPage:upperBound], len(pipelineVersions), npt, err
}

func (k *PipelineStoreKubernetes) DeletePipelineVersion(pipelineVersionId string) error {
	k8sPipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionId)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundError:") {
			return nil
		}
	}

	err = k.client.Delete(context.TODO(), k8sPipelineVersion)
	if err != nil && !k8serrors.IsNotFound(err) {
		return util.NewInternalServerError(err, "Failed to delete the pipeline version")
	}

	return nil
}

func (k *PipelineStoreKubernetes) getK8sPipeline(pipelineId string) (*v2beta1.Pipeline, error) {
	pipelines := v2beta1.PipelineList{}

	// Be careful, the deep copy is disabled here to reduce memory allocations
	listOptions := []ctrlclient.ListOption{ctrlclient.UnsafeDisableDeepCopy}

	if !common.IsMultiUserMode() && common.GetPodNamespace() != "" {
		listOptions = append(listOptions, ctrlclient.InNamespace(common.GetPodNamespace()))
	}

	err := k.client.List(context.TODO(), &pipelines, listOptions...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to find the pipeline")
	}

	for _, k8sPipeline := range pipelines.Items {
		if string(k8sPipeline.UID) == pipelineId {
			return &k8sPipeline, nil
		}
	}

	return nil, util.NewResourceNotFoundError("Pipeline", pipelineId)
}

func (k *PipelineStoreKubernetes) getK8sPipelineVersions(ctx context.Context, pipelineId string) (*v2beta1.PipelineVersionList, error) {
	pipelineVersions := v2beta1.PipelineVersionList{}

	// Be careful, the deep copy is disabled here to reduce memory allocations
	listOptions := []ctrlclient.ListOption{ctrlclient.UnsafeDisableDeepCopy}

	if !common.IsMultiUserMode() && common.GetPodNamespace() != "" {
		listOptions = append(listOptions, ctrlclient.InNamespace(common.GetPodNamespace()))
	}

	if pipelineId != "" {
		listOptions = append(listOptions, ctrlclient.MatchingLabels{"pipelines.kubeflow.org/pipeline-id": pipelineId})
	}

	err := k.client.List(ctx, &pipelineVersions, listOptions...)
	if err != nil {
		return nil, util.NewInternalServerError(
			err, "Failed to find the pipeline version associated with this pipeline version",
		)
	}

	return &pipelineVersions, nil
}

func (k *PipelineStoreKubernetes) getK8sPipelineVersion(ctx context.Context, pipelineVersionId string) (*v2beta1.PipelineVersion, error) {
	pipelineVersions, err := k.getK8sPipelineVersions(ctx, "")
	if err != nil {
		return nil, err
	}

	for _, k8sPipelineVersion := range pipelineVersions.Items {
		if string(k8sPipelineVersion.UID) == pipelineVersionId {
			return &k8sPipelineVersion, nil
		}
	}

	return nil, util.NewResourceNotFoundError("PipelineVersion", pipelineVersionId)
}

func (k *PipelineStoreKubernetes) createPipelineVersionWithPipeline(ctx context.Context, pipeline *model.Pipeline, pipelineVersion *model.PipelineVersion) (*model.PipelineVersion, error) {
	k8sPipelineVersion, err := v2beta1.FromPipelineVersionModel(*pipeline, *pipelineVersion)
	if err != nil {
		return nil, util.NewBadRequestError(err, "Invalid pipeline spec")
	}

	pipelineVersionStatus := k8sPipelineVersion.Status.DeepCopy()

	glog.Infof(
		"Creating the pipeline version %s/%s in Kubernetes", k8sPipelineVersion.Namespace, k8sPipelineVersion.Name,
	)
	err = k.client.Create(ctx, k8sPipelineVersion)
	if k8serrors.IsAlreadyExists(err) {
		return nil, util.NewAlreadyExistError(
			"Failed to create a new pipeline version. The name %v already exists. Please specify a new name",
			pipelineVersion.Name,
		)
	} else if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create the pipeline version")
	}

	k8sPipelineVersion.Status = *pipelineVersionStatus

	err = k.client.Status().Update(ctx, k8sPipelineVersion)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to set the pipeline version status")
	}

	return k8sPipelineVersion.ToModel()
}
