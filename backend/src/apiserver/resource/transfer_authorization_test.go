// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type transferReviewClient struct {
	reviews      []*authzv1.SubjectAccessReview
	deny         bool
	denyResource string
}

func (c *transferReviewClient) Create(_ context.Context, review *authzv1.SubjectAccessReview, _ metav1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	c.reviews = append(c.reviews, review.DeepCopy())
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: !c.deny && review.Spec.ResourceAttributes.Resource != c.denyResource}}, nil
}
func TestTransferAuthorizationRequiresNamespacePermissions(t *testing.T) {
	for _, importing := range []bool{false, true} {
		name := "export"
		if importing {
			name = "import"
		}
		t.Run(name, func(t *testing.T) {
			store, manager, _ := initWithExperiment(t)
			defer store.Close()
			oldMulti, oldShared := viper.Get(common.MultiUserMode), viper.Get(common.MultiUserModeSharedReadAccess)
			viper.Set(common.MultiUserMode, true)
			viper.Set(common.MultiUserModeSharedReadAccess, true)
			t.Cleanup(func() {
				viper.Set(common.MultiUserMode, oldMulti)
				viper.Set(common.MultiUserModeSharedReadAccess, oldShared)
			})
			reviewer := &transferReviewClient{deny: true}
			manager.subjectAccessReviewClient = reviewer
			err := manager.AuthorizeTransfer(multiUserContext(), "team", importing)
			require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "%v", err)
			require.Len(t, reviewer.reviews, 1, "shared-read mode must not bypass transfer authorization")
			reviewer.deny = false
			reviewer.reviews = nil
			require.NoError(t, manager.AuthorizeTransfer(multiUserContext(), "team", importing))
			require.Len(t, reviewer.reviews, 10)
			seen := map[string]bool{}
			for _, review := range reviewer.reviews {
				attrs := review.Spec.ResourceAttributes
				require.Equal(t, "team", attrs.Namespace)
				require.Equal(t, common.RbacPipelinesGroup, attrs.Group)
				require.NotEmpty(t, review.Spec.User)
				seen[attrs.Resource+":"+attrs.Verb] = true
			}
			verb := "get"
			if importing {
				verb = "create"
			}
			for _, resource := range []string{"experiments", "pipelines", "runs", "jobs", "artifacts"} {
				require.True(t, seen[resource+":list"])
				require.True(t, seen[resource+":"+verb])
			}
			reviewer.denyResource = "artifacts"
			require.True(t, util.IsUserErrorCodeMatch(manager.AuthorizeTransfer(multiUserContext(), "team", importing), codes.PermissionDenied), "run/catalog permissions do not grant artifact access")
			reviewer.denyResource = ""
			reviewer.reviews = nil
			for _, namespace := range []string{"", "../other", "Other"} {
				err := manager.AuthorizeTransfer(multiUserContext(), namespace, importing)
				require.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument), "%v", err)
			}
			require.Empty(t, reviewer.reviews)
		})
	}
}
