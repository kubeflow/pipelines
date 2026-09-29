// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type authorizationStatusSAR struct {
	status authorizationv1.SubjectAccessReviewStatus
}

func (c authorizationStatusSAR) Create(context.Context, *authorizationv1.SubjectAccessReview, metav1.CreateOptions) (*authorizationv1.SubjectAccessReview, error) {
	return &authorizationv1.SubjectAccessReview{Status: c.status}, nil
}

func TestIsAuthorized_EvaluationErrorsFailClosedForOrdinaryResources(t *testing.T) {
	for _, mode := range []string{"enforce", "audit"} {
		for _, test := range []struct {
			name            string
			allowed, denied bool
			evaluationError string
			wantCode        codes.Code
		}{
			{name: "clean allow", allowed: true, wantCode: codes.OK},
			{name: "clean deny", denied: true, wantCode: codes.PermissionDenied},
			{name: "allow with evaluation error", allowed: true, evaluationError: "private-authorization-details", wantCode: codes.Internal},
			{name: "deny with evaluation error", denied: true, evaluationError: "private-authorization-details", wantCode: codes.Internal},
		} {
			t.Run(mode+"/"+test.name, func(t *testing.T) {
				configureWorkflowIdentityAuditTest(t, mode == "audit")
				store, manager, _ := initWithExperiment(t)
				defer store.Close()
				manager.subjectAccessReviewClient = authorizationStatusSAR{status: authorizationv1.SubjectAccessReviewStatus{
					Allowed: test.allowed, Denied: test.denied, EvaluationError: test.evaluationError,
				}}
				err := manager.IsAuthorized(multiUserContext(), &authorizationv1.ResourceAttributes{
					Namespace: "ns1", Verb: common.RbacResourceVerbCreate,
					Group: common.RbacPipelinesGroup, Resource: common.RbacResourceTypeRuns,
				})
				if test.wantCode == codes.OK {
					require.NoError(t, err)
					return
				}
				require.Error(t, err)
				assert.True(t, util.IsUserErrorCodeMatch(err, test.wantCode), "unexpected authorization error: %v", err)
				assert.NotContains(t, err.Error(), "private-authorization-details")
			})
		}
	}
}
