// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// AuthenticateRuntimeTask always verifies the projected runtime token, including
// single-user installations. Public API identity headers and broad API tokens
// cannot opt into the recovery protocol.
func (r *ResourceManager) AuthenticateRuntimeTask(ctx context.Context, runID string) error {
	_, _, err := r.authenticateRuntimeTask(ctx, runID)
	return err
}

// AuthorizeRuntimeTask uses the verified runtime service account for RBAC. It
// never resolves the identity again through public API authenticators.
func (r *ResourceManager) AuthorizeRuntimeTask(ctx context.Context, runID, verb string) error {
	identity, run, err := r.authenticateRuntimeTask(ctx, runID)
	if err != nil {
		return err
	}
	if !common.IsMultiUserMode() || common.IsMultiUserSharedReadMode() &&
		(verb == common.RbacResourceVerbGet || verb == common.RbacResourceVerbList) {
		return nil
	}
	return r.isAuthorizedForIdentity(ctx, identity, &authorizationv1.ResourceAttributes{
		Namespace: run.Namespace,
		Name:      run.K8SName,
		Verb:      verb,
		Group:     common.RbacPipelinesGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeRuns,
	})
}

func (r *ResourceManager) authenticateRuntimeTask(ctx context.Context, runID string) (string, *model.Run, error) {
	if runID == "" {
		return "", nil, util.NewInvalidInputError("Run ID is required")
	}
	if r.tokenReviewClient == nil {
		return "", nil, util.NewUnauthenticatedError(fmt.Errorf("runtime TokenReview client unavailable"), "Runtime task authentication is unavailable")
	}
	ctx = auth.WithRequestedRunID(ctx, runID)
	authenticator := auth.NewTokenReviewAuthenticator(common.AuthorizationBearerTokenHeader, common.AuthorizationBearerTokenPrefix, []string{common.GetTokenReviewAudience()}, r.tokenReviewClient)
	identity, err := authenticator.GetUserIdentity(ctx)
	if err != nil {
		return "", nil, err
	}
	principal, ok := auth.AuthenticatedPrincipalFromContext(ctx)
	if !ok || principal.Scope != auth.TokenScopeRun || principal.RunID != runID {
		return "", nil, util.NewUnauthenticatedError(fmt.Errorf("token is not bound to this run"), "Runtime tasks require a run-scoped Kubernetes token")
	}
	run, err := r.GetRunWithHydration(runID, false)
	if err != nil {
		return "", nil, err
	}
	if !strings.HasPrefix(identity, "system:serviceaccount:"+run.Namespace+":") {
		return "", nil, util.NewUnauthenticatedError(fmt.Errorf("runtime service account namespace does not match"), "Runtime token must belong to the run namespace")
	}
	return identity, run, nil
}

func (r *ResourceManager) FinalizeStoppedDriver(runID string, generation int64, taskName, parentTaskID string, iterationIndex *int64) error {
	return r.taskStore.FinalizeStoppedDriver(runID, generation, taskName, parentTaskID, iterationIndex)
}
