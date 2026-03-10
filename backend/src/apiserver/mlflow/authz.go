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

package mlflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AuthorizationEnabledConfigKey is the Viper key that controls whether
	// the MLflow plugin performs a SubjectAccessReview in standalone mode.
	// In multi-user mode the check is always performed regardless of this key.
	AuthorizationEnabledConfigKey = "plugins.mlflow.authorizationEnabled"

	// mlflowAPIGroup is the Kubernetes API group used for MLflow RBAC.
	mlflowAPIGroup = "mlflow.kubeflow.org"
	// mlflowResource is the resource name used in SubjectAccessReview.
	mlflowResource = "experiments"
)

// AuthorizeExperimentAction checks whether the caller is permitted to perform
// MLflow experiment operations in the given namespace.
//
// In multi-user mode, the isAuthorized callback (typically ResourceManager.IsAuthorized)
// is used. It extracts the user identity from the context and performs a
// SubjectAccessReview using the caller's identity.
//
// In standalone mode, authorization is only enforced when AuthorizationEnabledConfigKey
// is set to "true". When enabled, the user identity is read from the forwarded
// header (KUBEFLOW_USERID_HEADER) and a SubjectAccessReview is created using
// the sarClient.
func AuthorizeExperimentAction(
	ctx context.Context,
	namespace string,
	sarClient client.SubjectAccessReviewInterface,
	isAuthorized func(context.Context, *authorizationv1.ResourceAttributes) error,
) error {
	resourceAttrs := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      "create",
		Group:     mlflowAPIGroup,
		Resource:  mlflowResource,
	}

	if common.IsMultiUserMode() {
		// Multi-user mode: delegate to the standard IsAuthorized path which
		// performs a SubjectAccessReview using the caller's identity.
		glog.Infof("MLflow authz: multi-user mode, delegating to IsAuthorized for namespace %s", namespace)
		return isAuthorized(ctx, resourceAttrs)
	}

	// Standalone mode: only enforce if authorizationEnabled is true.
	if !viper.GetBool(AuthorizationEnabledConfigKey) {
		glog.Info("MLflow authz: standalone mode, authorization disabled — skipping check")
		return nil
	}

	// Standalone mode with authorization enabled: read user identity from
	// forwarded headers and perform a SubjectAccessReview.
	userIdentity, err := getForwardedUserIdentity(ctx)
	if err != nil {
		return err
	}

	glog.Infof("MLflow authz: standalone mode, performing SAR for user %s in namespace %s", userIdentity, namespace)
	result, err := sarClient.Create(
		ctx,
		&authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				ResourceAttributes: resourceAttrs,
				User:               userIdentity,
			},
		},
		v1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create SubjectAccessReview for MLflow experiment action: %w", err)
	}
	if !result.Status.Allowed {
		return fmt.Errorf("not authorized: user %q is not permitted to perform MLflow experiment actions in namespace %q: %s",
			userIdentity, namespace, result.Status.Reason)
	}
	return nil
}

// getForwardedUserIdentity extracts the user identity from the incoming gRPC
// metadata using the configured KUBEFLOW_USERID_HEADER. This is used in
// standalone mode with kube-rbac-proxy, where the proxy forwards the
// authenticated user identity in a request header.
func getForwardedUserIdentity(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("User identity not found: no gRPC metadata in context")
	}

	headerKey := strings.ToLower(common.GetKubeflowUserIDHeader())
	values := md.Get(headerKey)
	if len(values) == 0 || values[0] == "" {
		return "", fmt.Errorf("User identity not found: header %q is missing or empty", headerKey)
	}

	identity := values[0]
	prefix := common.GetKubeflowUserIDPrefix()
	if prefix != "" && strings.HasPrefix(identity, prefix) {
		identity = strings.TrimPrefix(identity, prefix)
	}

	return identity, nil
}
