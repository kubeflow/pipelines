// Copyright 2026 The Kubeflow Authors
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

package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// 12 rows tagged by tagsFor: team=ml on i%4 in {0,1} gives 6 rows, and
// team=ml with env=prod on i%4 == 0 gives 3 rows.
const seedRows = 12

func tagsFor(i int) map[string]string {
	switch i % 4 {
	case 0:
		return map[string]string{"team": "ml", "env": "prod"}
	case 1:
		return map[string]string{"team": "ml", "env": "dev"}
	case 2:
		return nil
	default:
		return map[string]string{"team": "other", "env": "prod"}
	}
}

// filterSpec builds a URL-encoded v2 filter from raw predicate JSON objects.
func filterSpec(predicates ...string) string {
	return url.QueryEscape(`{"predicates":[` + strings.Join(predicates, ",") + `]}`)
}

func tagEquals(key, value string) string {
	return fmt.Sprintf(`{"key":%q,"operation":"EQUALS","string_value":%q}`, "tags."+key, value)
}

func nameNotEquals(value string) string {
	return fmt.Sprintf(`{"key":"name","operation":"NOT_EQUALS","string_value":%q}`, value)
}

// seedTaggedPipelines returns a pipeline server backed by the SQL store with
// seedRows pipelines and seedRows versions of the first pipeline, tagged by tagsFor.
func seedTaggedPipelines(t *testing.T) (*PipelineServer, string) {
	t.Helper()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	clientManager.UpdateUUID(util.NewUUIDGenerator())
	t.Cleanup(func() { clientManager.Close() })
	store := clientManager.PipelineStore()
	var parentID string
	for i := 0; i < seedRows; i++ {
		name := fmt.Sprintf("p-%02d", i)
		p, err := store.CreatePipeline(&model.Pipeline{Name: name, DisplayName: name, Status: model.PipelineReady, Tags: tagsFor(i)})
		require.NoError(t, err)
		if i == 0 {
			parentID = p.UUID
		}
	}
	for i := 0; i < seedRows; i++ {
		name := fmt.Sprintf("v-%02d", i)
		_, err := store.CreatePipelineVersion(&model.PipelineVersion{
			Name: name, DisplayName: name, PipelineId: parentID, Status: model.PipelineVersionReady,
			PipelineSpec: model.LargeText(v2SpecHelloWorld), Tags: tagsFor(i),
		})
		require.NoError(t, err)
	}
	rm := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	return createPipelineServer(rm, nil), parentID
}

// seedTaggedK8sPipelines returns a pipeline server backed by the Kubernetes
// pipeline store with seedRows Pipeline objects tagged by tagsFor.
func seedTaggedK8sPipelines(t *testing.T) *PipelineServer {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v2beta1.AddToScheme(scheme))
	var objects []ctrlclient.Object
	for i := 0; i < seedRows; i++ {
		name := fmt.Sprintf("p-%02d", i)
		objects = append(objects, &v2beta1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				UID:               types.UID(fmt.Sprintf("00000000-0000-0000-0000-0000000000%02d", i)),
				Name:              name,
				Namespace:         installationNamespace,
				CreationTimestamp: metav1.NewTime(time.Unix(int64(i+1), 0)),
			},
			Spec: v2beta1.PipelineSpec{DisplayName: name, Tags: tagsFor(i)},
		})
	}
	k8sClient := ctrlfake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	t.Cleanup(func() { clientManager.Close() })
	clientManager.SetPipelineStore(storage.NewPipelineStoreKubernetes(k8sClient, k8sClient))
	rm := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	return createPipelineServer(rm, nil)
}

type listPage struct {
	names     []string
	tags      []map[string]string
	totalSize int32
	next      string
}

// pager issues one list request through the real server handler.
type pager func(token string, size int32, filter string) (listPage, error)

func pipelinesPager(s *PipelineServer) pager {
	return func(token string, size int32, filter string) (listPage, error) {
		resp, err := s.ListPipelines(context.Background(), &apiv2.ListPipelinesRequest{PageToken: token, PageSize: size, Filter: filter})
		if err != nil {
			return listPage{}, err
		}
		pg := listPage{totalSize: resp.TotalSize, next: resp.NextPageToken}
		for _, p := range resp.Pipelines {
			pg.names = append(pg.names, p.DisplayName)
			pg.tags = append(pg.tags, p.Tags)
		}
		return pg, nil
	}
}

func versionsPager(s *PipelineServer, pipelineID string) pager {
	return func(token string, size int32, filter string) (listPage, error) {
		resp, err := s.ListPipelineVersions(context.Background(), &apiv2.ListPipelineVersionsRequest{
			PipelineId: pipelineID, PageToken: token, PageSize: size, Filter: filter,
		})
		if err != nil {
			return listPage{}, err
		}
		pg := listPage{totalSize: resp.TotalSize, next: resp.NextPageToken}
		for _, v := range resp.PipelineVersions {
			pg.names = append(pg.names, v.DisplayName)
			pg.tags = append(pg.tags, v.Tags)
		}
		return pg, nil
	}
}

// pagers returns every endpoint and pipeline-store backend under test.
func pagers(t *testing.T) map[string]pager {
	s, parentID := seedTaggedPipelines(t)
	return map[string]pager{
		"ListPipelines":        pipelinesPager(s),
		"ListPipelineVersions": versionsPager(s, parentID),
		"ListPipelines/k8s":    pipelinesPager(seedTaggedK8sPipelines(t)),
	}
}

// walk fetches page one with filter, then follows next_page_token sending
// laterFilter. An empty laterFilter is a token-only request, which the API's
// page-token mismatch error tells clients to send.
func walk(t *testing.T, get pager, size int32, filter, laterFilter string, rewriteToken func(string) string) ([]string, []listPage) {
	t.Helper()
	pg, err := get("", size, filter)
	require.NoError(t, err)
	pages := []listPage{pg}
	names := append([]string{}, pg.names...)
	for pg.next != "" && len(pages) <= seedRows {
		token := pg.next
		if rewriteToken != nil {
			token = rewriteToken(token)
		}
		pg, err = get(token, size, laterFilter)
		require.NoError(t, err)
		pages = append(pages, pg)
		names = append(names, pg.names...)
	}
	for i, p := range pages {
		t.Logf("page %d page_size=%d total_size=%d names=%v tags=%v", i+1, size, p.totalSize, p.names, p.tags)
	}
	return names, pages
}

// assertAllMatch checks every row on every page against want, that total_size
// is stable, and that wantCount distinct rows were returned.
func assertAllMatch(t *testing.T, pages []listPage, want map[string]string, wantCount int) {
	t.Helper()
	seen := map[string]bool{}
	for i, p := range pages {
		assert.Equal(t, int32(wantCount), p.totalSize, "page %d total_size", i+1)
		for j, tags := range p.tags {
			for k, v := range want {
				assert.Equal(t, v, tags[k], "page %d returned %q (tags=%v) outside filter %v", i+1, p.names[j], tags, want)
			}
			assert.False(t, seen[p.names[j]], "page %d repeated %q", i+1, p.names[j])
			seen[p.names[j]] = true
		}
	}
	assert.Len(t, seen, wantCount, "distinct rows across all pages")
}

func TestListWithTagFilter_TokenOnlyPagination(t *testing.T) {
	cases := map[string]struct {
		filter string
		want   map[string]string
		count  int
	}{
		"single": {filterSpec(tagEquals("team", "ml")), map[string]string{"team": "ml"}, 6},
		"multi":  {filterSpec(tagEquals("team", "ml"), tagEquals("env", "prod")), map[string]string{"team": "ml", "env": "prod"}, 3},
	}
	for name, get := range pagers(t) {
		for cname, c := range cases {
			// Ordering reference: the same query in one page.
			ordered, _ := walk(t, get, 100, c.filter, "", nil)
			for _, size := range []int32{1, 2, 4, 5} {
				t.Run(fmt.Sprintf("%s/%s/page_size=%d", name, cname, size), func(t *testing.T) {
					names, pages := walk(t, get, size, c.filter, "", nil)
					assertAllMatch(t, pages, c.want, c.count)
					assert.Equal(t, ordered, names, "order across pages")
				})
			}
		}
	}
}

func TestListWithTagFilter_RepeatedFilterPagination(t *testing.T) {
	for name, get := range pagers(t) {
		t.Run(name, func(t *testing.T) {
			f := filterSpec(tagEquals("team", "ml"))
			_, pages := walk(t, get, 2, f, f, nil)
			assertAllMatch(t, pages, map[string]string{"team": "ml"}, 6)
		})
	}
}

func TestListWithTagFilter_MixedNameAndTagFilterTokenOnly(t *testing.T) {
	for name, get := range pagers(t) {
		t.Run(name, func(t *testing.T) {
			// The non-tag predicate stays in the token's SQL filter; the tag predicate must survive too.
			_, pages := walk(t, get, 2, filterSpec(nameNotEquals("nope"), tagEquals("team", "ml")), "", nil)
			assertAllMatch(t, pages, map[string]string{"team": "ml"}, 6)
		})
	}
}

func TestListWithTagFilter_MismatchedFilterWithToken(t *testing.T) {
	cases := map[string]string{
		"different tag value":   filterSpec(tagEquals("team", "other")),
		"extra tag predicate":   filterSpec(tagEquals("team", "ml"), tagEquals("env", "prod")),
		"different name filter": filterSpec(nameNotEquals("other"), tagEquals("team", "ml")),
	}
	for name, get := range pagers(t) {
		for cname, later := range cases {
			t.Run(name+"/"+cname, func(t *testing.T) {
				pg, err := get("", 2, filterSpec(nameNotEquals("nope"), tagEquals("team", "ml")))
				require.NoError(t, err)
				require.NotEmpty(t, pg.next)
				_, err = get(pg.next, 2, later)
				require.Error(t, err, "a filter that differs from the page token's must be rejected")
				assert.Contains(t, err.Error(), "page token does not match")
			})
		}
	}
}

func TestListWithTagFilter_EmptyResult(t *testing.T) {
	for name, get := range pagers(t) {
		t.Run(name, func(t *testing.T) {
			pg, err := get("", 2, filterSpec(tagEquals("team", "absent")))
			require.NoError(t, err)
			assert.Empty(t, pg.names)
			assert.Equal(t, int32(0), pg.totalSize)
			assert.Empty(t, pg.next)
		})
	}
}

func TestListWithoutTagFilter_TokenOnlyUnchanged(t *testing.T) {
	for name, get := range pagers(t) {
		t.Run(name, func(t *testing.T) {
			ordered, _ := walk(t, get, 100, "", "", nil)
			names, pages := walk(t, get, 5, "", "", nil)
			assert.Len(t, names, seedRows)
			assert.Len(t, pages, 3)
			assert.Equal(t, ordered, names)
		})
	}
}

// stripTagFilters rewrites a page token into the layout issued before tag
// filters were stored in tokens.
func stripTagFilters(t *testing.T) func(string) string {
	return func(tok string) string {
		b, err := base64.StdEncoding.DecodeString(tok)
		require.NoError(t, err)
		m := map[string]interface{}{}
		require.NoError(t, json.Unmarshal(b, &m))
		delete(m, "TagFilters")
		b, err = json.Marshal(m)
		require.NoError(t, err)
		return base64.StdEncoding.EncodeToString(b)
	}
}

func TestListWithTagFilter_LegacyTokenWithRepeatedFilter(t *testing.T) {
	for name, get := range pagers(t) {
		t.Run(name, func(t *testing.T) {
			f := filterSpec(tagEquals("team", "ml"))
			_, pages := walk(t, get, 2, f, f, stripTagFilters(t))
			assertAllMatch(t, pages, map[string]string{"team": "ml"}, 6)
		})
	}
}
