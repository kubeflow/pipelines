package storage

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
)

const pollTimeout = 3 * time.Second

var (
	ErrUnsupportedField = errors.New("the field is unsupported")

	errDefaultVersionUnresolved = errors.New("spec.defaultVersionName does not resolve to exactly one pipeline version")
)

type PipelineStoreKubernetes struct {
	client        ctrlclient.Client
	clientNoCache ctrlclient.Client
}

func NewPipelineStoreKubernetes(k8sClient ctrlclient.Client, k8sClientNoCache ctrlclient.Client) *PipelineStoreKubernetes {
	return &PipelineStoreKubernetes{client: k8sClient, clientNoCache: k8sClientNoCache}
}

func (k *PipelineStoreKubernetes) GetPipelineByNameAndNamespace(name string, namespace string) (*model.Pipeline, error) {
	if namespace == "" {
		// The pod namespace is where KFP itself runs; falling back to it would cross tenants.
		if common.IsMultiUserMode() {
			return nil, util.NewInvalidInputError(
				"A namespace is required to look up pipeline %v in multi-user mode", name,
			)
		}

		namespace = common.GetPodNamespace()
	}

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

func (k *PipelineStoreKubernetes) ListPipelines(filterContext *model.FilterContext, opts *list.Options, tagFilters ...map[string]string) ([]*model.Pipeline, int, string, error) {
	var resolvedTagFilters map[string]string
	if len(tagFilters) > 0 {
		resolvedTagFilters = tagFilters[0]
	}
	k8sPipelines := v2beta1.PipelineList{}

	listOptions := []ctrlclient.ListOption{ctrlclient.UnsafeDisableDeepCopy}

	if filterContext.ReferenceKey != nil && filterContext.Type == model.NamespaceResourceType {
		listOptions = append(listOptions, ctrlclient.InNamespace(filterContext.ID))
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
		if opts.Filter != nil {
			found, err1 := opts.Filter.FilterK8sPipelines(k8sPipeline)
			if err1 != nil {
				return nil, 0, "", err1
			}
			if !found {
				continue
			}
		}
		// Filter by tags if tag filters are provided
		if len(resolvedTagFilters) > 0 {
			match := true
			for key, value := range resolvedTagFilters {
				if k8sPipeline.Spec.Tags[key] != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		pipelines = append(pipelines, k8sPipeline.ToModel())
	}

	return sortAndPaginate(pipelines, opts)
}

// sortAndPaginate orders items by the requested field and then by primary key,
// as the SQL stores do, and returns the page that opts points at. The cache
// returns objects in no particular order, so the key tiebreak is what keeps
// pages consistent between calls. The page starts at the first item at or after
// the token's position, so it does not depend on that item still existing.
func sortAndPaginate[T list.Listable](items []T, opts *list.Options) ([]T, int, string, error) {
	key := func(item T) string {
		return fmt.Sprint(item.GetFieldValue(item.PrimaryKeyColumnName()))
	}
	sortValue := func(item T) interface{} {
		if opts.SortByFieldName == "" {
			return nil
		}
		return item.GetFieldValue(opts.SortByFieldName)
	}
	// position compares an item with a (sort value, key) pair in list order.
	position := func(item T, otherSortValue interface{}, otherKey string) int {
		order := compareSortValues(sortValue(item), otherSortValue)
		if order == 0 {
			order = strings.Compare(key(item), otherKey)
		}
		if opts.IsDesc {
			return -order
		}
		return order
	}

	sort.Slice(items, func(i, j int) bool {
		return position(items[i], sortValue(items[j]), key(items[j])) < 0
	})

	start := 0
	if opts.KeyFieldValue != nil && fmt.Sprint(opts.KeyFieldValue) != "" {
		tokenKey := fmt.Sprint(opts.KeyFieldValue)
		start = sort.Search(len(items), func(i int) bool {
			return position(items[i], opts.SortByFieldValue, tokenKey) >= 0
		})
	}

	end := start + opts.PageSize
	if end >= len(items) {
		return items[start:], len(items), "", nil
	}
	nextPageToken, err := opts.NextPageToken(items[end])
	return items[start:end], len(items), nextPageToken, err
}

// compareSortValues compares two sort field values: numbers numerically and
// everything else as case-insensitive strings. A page token carries numbers as
// float64 after its JSON round trip, so all numeric kinds compare as float64.
func compareSortValues(a, b interface{}) int {
	aNumber, aIsNumber := sortValueAsFloat(a)
	bNumber, bIsNumber := sortValueAsFloat(b)
	if aIsNumber && bIsNumber {
		return cmp.Compare(aNumber, bNumber)
	}
	return strings.Compare(sortValueAsString(a), sortValueAsString(b))
}

func sortValueAsFloat(value interface{}) (float64, bool) {
	v := reflect.ValueOf(value)
	switch {
	case !v.IsValid():
		return 0, false
	case v.CanInt():
		return float64(v.Int()), true
	case v.CanUint():
		return float64(v.Uint()), true
	case v.CanFloat():
		return v.Float(), true
	default:
		return 0, false
	}
}

func sortValueAsString(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.ToLower(fmt.Sprint(value))
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

	// Deep copy to avoid mutating the cache (getK8sPipeline uses UnsafeDisableDeepCopy).
	pipelineCopy := k8sPipeline.DeepCopy()

	err = k.client.Delete(context.TODO(), pipelineCopy)
	if err != nil && !k8serrors.IsNotFound(err) {
		return util.NewInternalServerError(err, "Failed to delete the pipeline")
	}

	return k.deleteWithTimeout(pipelineCopy.Namespace, pipelineCopy.Name, &v2beta1.Pipeline{})
}

func (k *PipelineStoreKubernetes) CreatePipelineAndPipelineVersion(pipeline *model.Pipeline, pipelineVersion *model.PipelineVersion) (*model.Pipeline, *model.PipelineVersion, error) {
	pipeline.UUID = ""
	pipelineVersion.UUID = ""

	if pipeline.Name != pipelineVersion.Name {
		if _, err := v2beta1.NewPipelineVersionName(pipeline.Name, pipelineVersion.Name); err != nil {
			return nil, nil, err
		}
	}

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

	// Validate the pipeline name is a valid Kubernetes resource name before sending to the API.
	// Use IsDNS1123Subdomain (not IsDNS1123Label) because K8s metadata.name allows dots.
	if errs := validation.IsDNS1123Subdomain(pipeline.Name); len(errs) > 0 {
		return nil, util.NewInvalidInputError(
			"Invalid pipeline name %q: %s. Use 'display_name' for human-readable labels",
			pipeline.Name, strings.Join(errs, "; "),
		)
	}

	k8sPipeline := v2beta1.FromPipelineModel(*pipeline)

	glog.Infof("Creating the pipeline %s/%s in Kubernetes", k8sPipeline.Namespace, k8sPipeline.Name)

	err := k.client.Create(context.TODO(), &k8sPipeline)
	if k8serrors.IsAlreadyExists(err) {
		return nil, util.NewAlreadyExistError(
			"Failed to create a new pipeline. The name %v already exists. Please specify a new name", pipeline.Name,
		)
	} else if k8serrors.IsInvalid(err) && strings.Contains(err.Error(), "metadata.name") {
		return nil, util.NewBadKubernetesNameError("pipeline")
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

	// Deep copy to avoid mutating the cache (getK8sPipelineVersion uses UnsafeDisableDeepCopy).
	versionCopy := k8sPipelineVersion.DeepCopy()

	conditionSet := false

	for i := range versionCopy.Status.Conditions {
		condition := &versionCopy.Status.Conditions[i]

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
		versionCopy.Status.Conditions = append(versionCopy.Status.Conditions, v2beta1.SimplifiedCondition{
			Type:    "PipelineVersionStatus",
			Reason:  string(status),
			Message: string(status),
			Status:  metav1.ConditionTrue,
		})
	}

	err = k.client.Status().Update(context.TODO(), versionCopy)
	if err != nil && k8serrors.IsConflict(err) {
		return k.UpdatePipelineVersionStatus(pipelineVersionId, status)
	} else if err != nil {
		return util.NewInternalServerError(err, "Failed to update the pipeline version status")
	}
	return k.updateWithTimeout(versionCopy)
}

func (k *PipelineStoreKubernetes) CreatePipelineVersion(pipelineVersion *model.PipelineVersion) (*model.PipelineVersion, error) {
	pipeline, err := k.GetPipeline(pipelineVersion.PipelineId)
	if err != nil {
		return nil, err
	}

	return k.createPipelineVersionWithPipeline(context.TODO(), pipeline, pipelineVersion)
}

// GetDefaultPipelineVersion returns spec.defaultVersionName when set, otherwise the newest version.
// A default that does not resolve is an error, not a fallback.
func (k *PipelineStoreKubernetes) GetDefaultPipelineVersion(pipelineID string) (*model.PipelineVersion, error) {
	k8sPipeline, err := k.getK8sPipeline(pipelineID)
	if err != nil {
		return nil, err
	}

	ownedVersions, err := k.ownedPipelineVersions(context.TODO(), k8sPipeline)
	if err != nil {
		return nil, err
	}

	if defaultVersionName := k8sPipeline.Spec.DefaultVersionName; defaultVersionName != "" {
		pipelineVersion, err := resolveDefaultPipelineVersion(ownedVersions, defaultVersionName)
		if err != nil {
			return nil, util.Wrapf(
				err,
				"Failed to resolve the default pipeline version %q of pipeline %v",
				defaultVersionName, pipelineID,
			)
		}

		return pipelineVersion, nil
	}

	var latestK8sPipelineVersion *v2beta1.PipelineVersion

	for _, k8sPipelineVersion := range ownedVersions {
		if latestK8sPipelineVersion == nil || isNewerPipelineVersion(k8sPipelineVersion, latestK8sPipelineVersion) {
			latestK8sPipelineVersion = k8sPipelineVersion
		}
	}

	if latestK8sPipelineVersion == nil {
		return nil, util.NewResourceNotFoundError("PipelineVersion", "Default")
	}

	return latestK8sPipelineVersion.ToModel()
}

// ownedPipelineVersions lists the pipeline's versions in its own namespace and keeps those its
// ownerReferences claim. The pipelines.kubeflow.org/pipeline-id label is user-mutable, so it is not
// used to select candidates: a stale label must neither hide an owned version nor surface a foreign one.
func (k *PipelineStoreKubernetes) ownedPipelineVersions(
	ctx context.Context, k8sPipeline *v2beta1.Pipeline,
) ([]*v2beta1.PipelineVersion, error) {
	k8sPipelineVersions, err := k.listPipelineVersionsInNamespace(ctx, k8sPipeline.Namespace)
	if err != nil {
		return nil, err
	}

	owned := make([]*v2beta1.PipelineVersion, 0, len(k8sPipelineVersions.Items))

	for i := range k8sPipelineVersions.Items {
		k8sPipelineVersion := &k8sPipelineVersions.Items[i]
		if k8sPipelineVersion.IsOwnedByPipeline(string(k8sPipeline.UID)) {
			owned = append(owned, k8sPipelineVersion)
		}
	}

	return owned, nil
}

// listPipelineVersionsInNamespace lists the pipeline version CRs in a namespace.
//
// Be careful, the deep copy is disabled here to reduce memory allocations.
// Callers that mutate the returned objects must deep copy them first.
func (k *PipelineStoreKubernetes) listPipelineVersionsInNamespace(
	ctx context.Context, namespace string,
) (*v2beta1.PipelineVersionList, error) {
	k8sPipelineVersions := &v2beta1.PipelineVersionList{}

	err := k.client.List(
		ctx, k8sPipelineVersions,
		ctrlclient.UnsafeDisableDeepCopy, ctrlclient.InNamespace(namespace),
	)
	if err != nil {
		return nil, util.NewInternalServerError(
			err, "Failed to list pipeline versions in namespace %v", namespace,
		)
	}

	return k8sPipelineVersions, nil
}

// GetAnyPipelineVersionID returns the id of one version owned by the pipeline, or "" if it owns
// none. It returns on the first owned CR and never calls ToModel, so a caller that only needs to
// know whether any version exists does not pay to parse the whole version history.
func (k *PipelineStoreKubernetes) GetAnyPipelineVersionID(pipelineID string) (string, error) {
	k8sPipeline, err := k.getK8sPipeline(pipelineID)
	if err != nil {
		return "", err
	}

	k8sPipelineVersions, err := k.listPipelineVersionsInNamespace(context.TODO(), k8sPipeline.Namespace)
	if err != nil {
		return "", err
	}

	for i := range k8sPipelineVersions.Items {
		if k8sPipelineVersions.Items[i].IsOwnedByPipeline(string(k8sPipeline.UID)) {
			return string(k8sPipelineVersions.Items[i].UID), nil
		}
	}

	return "", nil
}

// resolveDefaultPipelineVersion matches the pin on the name ToModel reports: spec.versionName, or
// metadata.name when unset.
func resolveDefaultPipelineVersion(
	k8sPipelineVersions []*v2beta1.PipelineVersion, defaultVersionName string,
) (*model.PipelineVersion, error) {
	var matches []*v2beta1.PipelineVersion

	for _, k8sPipelineVersion := range k8sPipelineVersions {
		versionName := k8sPipelineVersion.Spec.VersionName
		if versionName == "" {
			versionName = k8sPipelineVersion.Name
		}

		if versionName == defaultVersionName {
			matches = append(matches, k8sPipelineVersion)
		}
	}

	switch len(matches) {
	case 0:
		return nil, util.NewFailedPreconditionError(
			errDefaultVersionUnresolved,
			"no pipeline version is named %q; set spec.defaultVersionName to an existing version",
			defaultVersionName,
		)
	case 1:
		return matches[0].ToModel()
	default:
		return nil, util.NewFailedPreconditionError(
			errDefaultVersionUnresolved,
			"%d pipeline versions are named %q; spec.defaultVersionName must match exactly one",
			len(matches), defaultVersionName,
		)
	}
}

// isNewerPipelineVersion reports whether a should be preferred over b. CreationTimestamp has second
// granularity, so same-second versions tie; the UID breaks the tie so repeated calls agree.
func isNewerPipelineVersion(a, b *v2beta1.PipelineVersion) bool {
	aCreated, bCreated := a.CreationTimestamp.Time, b.CreationTimestamp.Time
	if !aCreated.Equal(bCreated) {
		return aCreated.After(bCreated)
	}

	return string(a.UID) > string(b.UID)
}

func (k *PipelineStoreKubernetes) GetPipelineVersion(pipelineVersionId string) (*model.PipelineVersion, error) {
	pipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionId)
	if err != nil {
		return nil, err
	}

	return pipelineVersion.ToModel()
}

// GetPipelineVersionByName returns a pipeline version by name under the given pipeline.
// It resolves the pipeline's namespace and name via getK8sPipeline, then performs a
// two-stage lookup: first by composite name ({pipelineName}-{versionName}), then by
// bare name (CRs created before composite naming, or managed via GitOps). Both stages
// verify ownership via OwnerReferences before returning.
func (k *PipelineStoreKubernetes) GetPipelineVersionByName(pipelineID, versionName string) (*model.PipelineVersion, error) {
	k8sPipeline, err := k.getK8sPipeline(pipelineID)
	if err != nil {
		return nil, err
	}

	return k.getPipelineVersionByNameInNamespace(k8sPipeline.Namespace, pipelineID, k8sPipeline.Name, versionName)
}

func (k *PipelineStoreKubernetes) getPipelineVersionByNameInNamespace(namespace, pipelineID, pipelineName, versionName string) (*model.PipelineVersion, error) {
	pipelineVersion := v2beta1.PipelineVersion{}

	// Try composite name first ({pipelineName}-{versionName})
	if pipelineName != "" {
		compositeName := pipelineName + "-" + versionName
		err := k.client.Get(context.TODO(), ctrlclient.ObjectKey{
			Namespace: namespace,
			Name:      compositeName,
		}, &pipelineVersion)
		if err == nil {
			if pipelineVersion.IsOwnedByPipeline(pipelineID) {
				return pipelineVersion.ToModel()
			}
			// Composite name exists but belongs to a different pipeline (hyphen
			// collision). Fall through to bare-name lookup.
		} else if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}

	// Fallback: try bare name (CRs created before composite naming, or managed via GitOps)
	err := k.client.Get(context.TODO(), ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      versionName,
	}, &pipelineVersion)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, util.NewResourceNotFoundError("PipelineVersion", versionName)
		}
		return nil, err
	}

	if !pipelineVersion.IsOwnedByPipeline(pipelineID) {
		return nil, util.NewResourceNotFoundError("PipelineVersion", versionName)
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

func (k *PipelineStoreKubernetes) ListPipelineVersions(pipelineID string, opts *list.Options, tagFilters ...map[string]string) (versions []*model.PipelineVersion, totalSize int, nextPageToken string, err error) {
	var resolvedTagFilters map[string]string
	if len(tagFilters) > 0 {
		resolvedTagFilters = tagFilters[0]
	}
	k8sPipelineVersions, err := k.getK8sPipelineVersions(context.TODO(), pipelineID, "")
	if err != nil {
		return nil, 0, "", err
	}

	pipelineVersions := make([]*model.PipelineVersion, 0, len(k8sPipelineVersions.Items))

	for _, k8sPipelineVersion := range k8sPipelineVersions.Items {
		if opts.Filter != nil {
			found, err1 := opts.Filter.FilterK8sPipelineVersions(k8sPipelineVersion)
			if err1 != nil {
				return nil, 0, "", err1
			}
			if !found {
				continue
			}
		}
		// Filter by tags if tag filters are provided
		if len(resolvedTagFilters) > 0 {
			match := true
			for key, value := range resolvedTagFilters {
				if k8sPipelineVersion.Spec.Tags[key] != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		pipelineVersion, err := k8sPipelineVersion.ToModel()
		if err != nil {
			return nil, 0, "", err
		}
		pipelineVersions = append(pipelineVersions, pipelineVersion)
	}

	return sortAndPaginate(pipelineVersions, opts)
}

func (k *PipelineStoreKubernetes) DeletePipelineVersion(pipelineVersionId string) error {
	k8sPipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionId)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundError:") {
			return nil
		}

		return err
	}

	// Deep copy to avoid mutating the cache (getK8sPipelineVersion uses UnsafeDisableDeepCopy).
	versionCopy := k8sPipelineVersion.DeepCopy()

	err = k.client.Delete(context.TODO(), versionCopy)
	if err != nil && !k8serrors.IsNotFound(err) {
		return util.NewInternalServerError(err, "Failed to delete the pipeline version")
	}

	return k.deleteWithTimeout(versionCopy.Namespace, versionCopy.Name, &v2beta1.PipelineVersion{})
}

// deleteWithTimeout polls until the given namespaced resource is NotFound or the timeout expires.
func (k *PipelineStoreKubernetes) deleteWithTimeout(namespace string, name string, exampleObject ctrlclient.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), pollTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// No need to return an error here, because the resource is deleted, it's just the cache hasn't updated yet.
			return nil
		default:
			err := k.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, exampleObject)
			if k8serrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return util.NewInternalServerError(err, "failed to check deletion status")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// updateWithTimeout polls the cache until the resource version matches the updated object or the timeout expires.
func (k *PipelineStoreKubernetes) updateWithTimeout(updated ctrlclient.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), pollTimeout)
	defer cancel()

	key := types.NamespacedName{Namespace: updated.GetNamespace(), Name: updated.GetName()}
	targetVersion := updated.GetResourceVersion()

	for {
		select {
		case <-ctx.Done():
			// Not fatal — the cache will eventually sync.
			return nil
		default:
			current := updated.DeepCopyObject().(ctrlclient.Object)
			err := k.client.Get(ctx, key, current)
			if err != nil {
				return nil
			}
			if current.GetResourceVersion() == targetVersion {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (k *PipelineStoreKubernetes) getK8sPipeline(pipelineId string) (*v2beta1.Pipeline, error) {
	pipelines := v2beta1.PipelineList{}

	// Be careful, the deep copy is disabled here to reduce memory allocations.
	// Callers that mutate the returned object must deep copy it first.
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

	// Be careful, the deep copy is disabled here to reduce memory allocations.
	// Callers that mutate the returned objects must deep copy them first.
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
		return nil, util.NewInternalServerError(err, "%s", errMsg)
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

	// Fallback to not using the cache
	err = k.clientNoCache.List(ctx, &pipelineVersions, listOptions...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "%s", errMsg)
	}

	if pipelineVersionId == "" {
		return &pipelineVersions, nil
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
		var userError *util.UserError
		if errors.As(err, &userError) {
			return nil, err
		}
		return nil, util.NewBadRequestError(err, "Invalid pipeline spec")
	}

	// Check for logical name collision (covers legacy bare-name CRs and composite-name CRs)
	if _, lookupErr := k.getPipelineVersionByNameInNamespace(pipeline.Namespace, pipeline.UUID, pipeline.Name, pipelineVersion.Name); lookupErr == nil {
		return nil, util.NewAlreadyExistError(
			"Failed to create a new pipeline version. The name %v already exists. Please specify a new name",
			pipelineVersion.Name,
		)
	} else if !util.IsUserErrorCodeMatch(lookupErr, codes.NotFound) {
		return nil, lookupErr
	}

	glog.Infof(
		"Creating the pipeline version %s/%s in Kubernetes", k8sPipelineVersion.Namespace, k8sPipelineVersion.Name,
	)
	err = k.client.Create(ctx, k8sPipelineVersion)
	if k8serrors.IsAlreadyExists(err) {
		return nil, util.NewAlreadyExistError(
			"Failed to create a new pipeline version. The name %v already exists (resource name: %v). Please specify a new name",
			pipelineVersion.Name, k8sPipelineVersion.Name,
		)
	} else if k8serrors.IsInvalid(err) && strings.Contains(err.Error(), "metadata.name") {
		return nil, util.NewBadKubernetesNameError("pipeline version")
	} else if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create the pipeline version")
	}

	return k8sPipelineVersion.ToModel()
}

func (k *PipelineStoreKubernetes) UpdatePipelineFields(pipelineID string, displayName string, tags map[string]string) error {
	k8sPipeline, err := k.getK8sPipeline(pipelineID)
	if err != nil {
		return err
	}
	// Deep copy to avoid mutating the cache (getK8sPipeline uses UnsafeDisableDeepCopy).
	pipelineCopy := k8sPipeline.DeepCopy()
	if displayName != "" {
		pipelineCopy.Spec.DisplayName = displayName
	}
	if tags != nil {
		pipelineCopy.Spec.Tags = tags
	}
	if err := k.client.Update(context.TODO(), pipelineCopy); err != nil {
		return util.NewInternalServerError(err, "Failed to update pipeline %v", pipelineID)
	}
	return k.updateWithTimeout(pipelineCopy)
}

func (k *PipelineStoreKubernetes) UpdatePipelineVersionFields(pipelineVersionID string, displayName string, tags map[string]string) error {
	k8sPipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionID)
	if err != nil {
		return err
	}
	// Deep copy to avoid mutating the cache (getK8sPipelineVersion uses UnsafeDisableDeepCopy).
	versionCopy := k8sPipelineVersion.DeepCopy()
	if displayName != "" {
		versionCopy.Spec.DisplayName = displayName
	}
	if tags != nil {
		versionCopy.Spec.Tags = tags
	}
	if err := k.client.Update(context.TODO(), versionCopy); err != nil {
		return util.NewInternalServerError(err, "Failed to update pipeline version %v", pipelineVersionID)
	}
	return k.updateWithTimeout(versionCopy)
}

// Pipeline tag operations

func (k *PipelineStoreKubernetes) CreateOrUpdatePipelineTags(pipelineID string, tags map[string]string) error {
	k8sPipeline, err := k.getK8sPipeline(pipelineID)
	if err != nil {
		return err
	}
	// Deep copy to avoid mutating the cache (getK8sPipeline uses UnsafeDisableDeepCopy).
	pipelineCopy := k8sPipeline.DeepCopy()
	pipelineCopy.Spec.Tags = tags
	if err := k.client.Update(context.TODO(), pipelineCopy); err != nil {
		return util.NewInternalServerError(err, "Failed to update tags for pipeline %v", pipelineID)
	}
	return k.updateWithTimeout(pipelineCopy)
}

func (k *PipelineStoreKubernetes) GetPipelineTags(pipelineID string) (map[string]string, error) {
	k8sPipeline, err := k.getK8sPipeline(pipelineID)
	if err != nil {
		return nil, err
	}
	return k8sPipeline.Spec.Tags, nil
}

// GetPipelineTagsForPipelines fetches tags for each pipeline individually because
// the K8s controller-runtime client does not provide a batch Get API. In practice,
// this method is not called from the K8s code path since ListPipelines already
// populates tags via ToModel().
func (k *PipelineStoreKubernetes) GetPipelineTagsForPipelines(pipelineIds []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	for _, id := range pipelineIds {
		tags, err := k.GetPipelineTags(id)
		if err != nil {
			return nil, err
		}
		if len(tags) > 0 {
			result[id] = tags
		}
	}
	return result, nil
}

func (k *PipelineStoreKubernetes) DeletePipelineTags(pipelineID string) error {
	return k.CreateOrUpdatePipelineTags(pipelineID, nil)
}

// Pipeline version tag operations

func (k *PipelineStoreKubernetes) CreateOrUpdatePipelineVersionTags(pipelineVersionID string, tags map[string]string) error {
	k8sPipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionID)
	if err != nil {
		return err
	}
	// Deep copy to avoid mutating the cache (getK8sPipelineVersion uses UnsafeDisableDeepCopy).
	versionCopy := k8sPipelineVersion.DeepCopy()
	versionCopy.Spec.Tags = tags
	if err := k.client.Update(context.TODO(), versionCopy); err != nil {
		return util.NewInternalServerError(err, "Failed to update tags for pipeline version %v", pipelineVersionID)
	}
	return k.updateWithTimeout(versionCopy)
}

func (k *PipelineStoreKubernetes) GetPipelineVersionTags(pipelineVersionID string) (map[string]string, error) {
	k8sPipelineVersion, err := k.getK8sPipelineVersion(context.TODO(), pipelineVersionID)
	if err != nil {
		return nil, err
	}
	return k8sPipelineVersion.Spec.Tags, nil
}

func (k *PipelineStoreKubernetes) GetPipelineVersionTagsForVersions(pipelineVersionIds []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	for _, id := range pipelineVersionIds {
		tags, err := k.GetPipelineVersionTags(id)
		if err != nil {
			return nil, err
		}
		if len(tags) > 0 {
			result[id] = tags
		}
	}
	return result, nil
}

func (k *PipelineStoreKubernetes) DeletePipelineVersionTags(pipelineVersionID string) error {
	return k.CreateOrUpdatePipelineVersionTags(pipelineVersionID, nil)
}
