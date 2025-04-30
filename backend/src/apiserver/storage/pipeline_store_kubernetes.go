package storage

import (
	"context"
	"fmt"
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

	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
)

var (
	ErrNoV1             = errors.New("the v1 API is not available for the Kubernetes pipeline store")
	ErrUnsupportedField = errors.New("the field is unsupported")
)

type PipelineStoreKubernetes struct {
	client        ctrlclient.Client
	clientNoCache ctrlclient.Client
}

func NewPipelineStoreKubernetes(k8sClient ctrlclient.Client, k8sClientNoCache ctrlclient.Client) *PipelineStoreKubernetes {
	return &PipelineStoreKubernetes{client: k8sClient, clientNoCache: k8sClientNoCache}
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
		}
		found, err1 := opts.Filter.FilterK8sPipelines(k8sPipeline)
		if err1 != nil {
			return nil, 0, "", err1
		}
		if found {
			pipelines = append(pipelines, k8sPipeline.ToModel())
		}
	}

	// Because controller-client does not have sorting, use this function to sort by fields.
	sort.SliceStable(pipelines, func(i, j int) bool {
		if opts.SortByFieldName != "" {
			elementA := pipelines[i].GetFieldValue(opts.SortByFieldName)
			elementB := pipelines[j].GetFieldValue(opts.SortByFieldName)

			if elementA != nil && elementB != nil {
				switch elementA.(type) {
				case int64:
					return elementA.(int64) > elementB.(int64)
				case float64:
					return elementA.(float64) > elementB.(float64)
				case string:
					return elementA.(string) > elementB.(string)
				default:
					glog.Warningf("Field type %T in %s not recognized. Sorting will not work.", elementA, opts.SortByFieldName)
					return false
				}
			}
		}
		return false
	})
	if !opts.IsDesc {
		slices.Reverse(pipelines)
	}

	// If there's pagination, find the index of the first element of the next page
	nextPage := 0
	for p, pipeline := range pipelines {
		if opts.KeyFieldValue != "" {
			if strings.Compare(fmt.Sprint(opts.KeyFieldValue), pipeline.UUID) == 0 {
				nextPage = p
				break
			}
		}
	}

	// Split results on multiple pages if needed
	upperBound := nextPage + opts.PageSize
	npt := ""
	if len(pipelines) > upperBound {
		npt, err = opts.NextPageToken(pipelines[upperBound])
	} else {
		upperBound = len(pipelines)
	}
	return pipelines[nextPage:upperBound], len(pipelines), npt, err
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
		return err
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
	k8sPipelineVersions, err := k.getK8sPipelineVersions(context.TODO(), pipelineId, "")
	if err != nil {
		return nil, err
	}

	var latestK8sPipelineVersion *v2beta1.PipelineVersion

	for _, k8sPipelineVersion := range k8sPipelineVersions.Items {
		if latestK8sPipelineVersion == nil || k8sPipelineVersion.CreationTimestamp.Time.After(latestK8sPipelineVersion.CreationTimestamp.Time) {
			latestK8sPipelineVersion = &k8sPipelineVersion
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
	pipelineVersion := v2beta1.PipelineVersion{}

	if common.GetPodNamespace() == "" {
		return nil, fmt.Errorf("Error returning the pod namespace. Ensure you have POD_NAMESPACE environment variable set in the API Server pod.")
	}

	err := k.client.Get(context.TODO(), ctrlclient.ObjectKey{
		Namespace: common.GetPodNamespace(),
		Name:      name,
	}, &pipelineVersion)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, util.NewResourceNotFoundError("PipelineVersion", name)
		}
		return nil, err
	}

	return pipelineVersion.ToModel()
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
	k8sPipelineVersions, err := k.getK8sPipelineVersions(context.TODO(), pipelineId, "")
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
	sort.SliceStable(pipelineVersions, func(i, j int) bool {
		if opts.SortByFieldName != "" {
			elementA := pipelineVersions[i].GetFieldValue(opts.SortByFieldName)
			elementB := pipelineVersions[j].GetFieldValue(opts.SortByFieldName)

			if elementA != nil && elementB != nil {
				switch elementA.(type) {
				case int64:
					return elementA.(int64) > elementB.(int64)
				case float64:
					return elementA.(float64) > elementB.(float64)
				case string:
					return elementA.(string) > elementB.(string)
				default:
					glog.Warningf("Field type %T in %s not recognized. Sorting will not work.", elementA, opts.SortByFieldName)
					return false
				}
			}
		}
		return false
	})
	if !opts.IsDesc {
		slices.Reverse(pipelineVersions)
	}

	// If there's pagination, find the index of the first element of the next page
	nextPage := 0
	if opts.KeyFieldValue != "" {
		for p, pipelineVersion := range pipelineVersions {
			if strings.Compare(fmt.Sprint(opts.KeyFieldValue), pipelineVersion.UUID) == 0 {
				nextPage = p
				break
			}
		}
	}

	// Split results on multiple pages if needed
	upperBound := nextPage + opts.PageSize
	npt := ""
	if len(pipelineVersions) > upperBound {
		npt, err = opts.NextPageToken(pipelineVersions[upperBound])
	} else {
		upperBound = len(pipelineVersions)
	}
	return pipelineVersions[nextPage:upperBound], len(pipelineVersions), npt, err
}

func (k *PipelineStoreKubernetes) DeletePipelineVersion(pipelineVersionId string) error {
	k8sPipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionId)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundError:") {
			return nil
		}

		return err
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
		return nil, util.NewInternalServerError(err, "Failed to list the pipelines")
	}

	for _, k8sPipeline := range pipelines.Items {
		if string(k8sPipeline.UID) == pipelineId {
			return &k8sPipeline, nil
		}
	}

	// Fallback to not using the cache
	err = k.clientNoCache.List(context.TODO(), &pipelines, listOptions...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list the pipelines")
	}

	for _, k8sPipeline := range pipelines.Items {
		if string(k8sPipeline.UID) == pipelineId {
			return &k8sPipeline, nil
		}
	}

	return nil, util.NewResourceNotFoundError("Pipeline", pipelineId)
}

func (k *PipelineStoreKubernetes) getK8sPipelineVersions(
	ctx context.Context, pipelineId string, pipelineVersionId string,
) (*v2beta1.PipelineVersionList, error) {
	pipelineVersions := v2beta1.PipelineVersionList{}

	// Be careful, the deep copy is disabled here to reduce memory allocations
	listOptions := []ctrlclient.ListOption{ctrlclient.UnsafeDisableDeepCopy}

	if !common.IsMultiUserMode() && common.GetPodNamespace() != "" {
		listOptions = append(listOptions, ctrlclient.InNamespace(common.GetPodNamespace()))
	}

	var errMsg string
	if pipelineVersionId != "" {
		errMsg = "Failed to get the pipeline version with ID " + pipelineVersionId
	} else {
		errMsg = "Failed to list pipeline versions"
	}

	if pipelineId != "" {
		errMsg += " associated with the pipeline with ID " + pipelineId
		listOptions = append(listOptions, ctrlclient.MatchingLabels{"pipelines.kubeflow.org/pipeline-id": pipelineId})
	}

	err := k.client.List(ctx, &pipelineVersions, listOptions...)
	if err != nil {
		return nil, util.NewInternalServerError(err, errMsg)
	}

	// If there is no pipeline version ID filter, then just return the results
	if pipelineVersionId == "" {
		return &pipelineVersions, nil
	}

	for _, pipelineVersion := range pipelineVersions.Items {
		if string(pipelineVersion.UID) == pipelineVersionId {
			return &v2beta1.PipelineVersionList{Items: []v2beta1.PipelineVersion{pipelineVersion}}, nil
		}
	}

	// Fallback to not using the cache if the specific pipeline version is missing
	err = k.clientNoCache.List(ctx, &pipelineVersions, listOptions...)
	if err != nil {
		return nil, util.NewInternalServerError(err, errMsg)
	}

	for _, pipelineVersion := range pipelineVersions.Items {
		if string(pipelineVersion.UID) == pipelineVersionId {
			return &v2beta1.PipelineVersionList{Items: []v2beta1.PipelineVersion{pipelineVersion}}, nil
		}
	}

	return nil, util.NewResourceNotFoundError("PipelineVersion", pipelineVersionId)
}

func (k *PipelineStoreKubernetes) getK8sPipelineVersion(ctx context.Context, pipelineVersionId string) (*v2beta1.PipelineVersion, error) {
	pipelineVersions, err := k.getK8sPipelineVersions(ctx, "", pipelineVersionId)
	if err != nil {
		return nil, err
	}

	return &pipelineVersions.Items[0], nil
}

func (k *PipelineStoreKubernetes) createPipelineVersionWithPipeline(ctx context.Context, pipeline *model.Pipeline, pipelineVersion *model.PipelineVersion) (*model.PipelineVersion, error) {
	k8sPipelineVersion, err := v2beta1.FromPipelineVersionModel(*pipeline, *pipelineVersion)
	if err != nil {
		return nil, util.NewBadRequestError(err, "Invalid pipeline spec")
	}

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

	return k8sPipelineVersion.ToModel()
}
