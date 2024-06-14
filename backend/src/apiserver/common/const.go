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

package common

const (
	RbacKubeflowGroup    = "kubeflow.org"
	RbacPipelinesGroup   = "pipelines.kubeflow.org"
	RbacPipelinesVersion = "v1beta1"

	RbacResourceTypePipelines          = "pipelines"
	RbacResourceTypeExperiments        = "experiments"
	RbacResourceTypeRuns               = "runs"
	RbacResourceTypeJobs               = "jobs"
	RbacResourceTypeViewers            = "viewers"
	RbacResourceTypeVisualizations     = "visualizations"
	RbacResourceTypeScheduledWorkflows = "scheduledworkflows"
	RbacResourceTypeWorkflows          = "workflows"
	RbacResourceTypeArtifacts          = "artifacts"

	RbacResourceVerbArchive       = "archive"
	RbacResourceVerbUpdate        = "update"
	RbacResourceVerbCreate        = "create"
	RbacResourceVerbDelete        = "delete"
	RbacResourceVerbDisable       = "disable"
	RbacResourceVerbEnable        = "enable"
	RbacResourceVerbGet           = "get"
	RbacResourceVerbList          = "list"
	RbacResourceVerbRetry         = "retry"
	RbacResourceVerbTerminate     = "terminate"
	RbacResourceVerbUnarchive     = "unarchive"
	RbacResourceVerbReportMetrics = "reportMetrics"
	RbacResourceVerbReadArtifact  = "readArtifact"
	RbacResourceVerbReport        = "report"
)

const (
	GoogleIAPUserIdentityHeader    string = "x-goog-authenticated-user-email"
	GoogleIAPUserIdentityPrefix    string = "accounts.google.com:"
	AuthorizationBearerTokenHeader string = "Authorization"
	AuthorizationBearerTokenPrefix string = "Bearer "
)

const DefaultTokenReviewAudience string = "pipelines.kubeflow.org"

const (
	DefaultMetadataGrpcServiceServiceHost = "metadata-grpc-service"
	DefaultMetadataGrpcServiceServicePort = "8080"
)

const (
	DefaultPipelineRunnerServiceAccount = "pipeline-runner"
	HasDefaultBucketEnvVar              = "HAS_DEFAULT_BUCKET"
	DefaultBucketNameEnvVar             = "BUCKET_NAME"
	ProjectIDEnvVar                     = "PROJECT_ID"
)

const DefaultSignedURLExpiryTimeSeconds = 15

const (
	MaxFileNameLength = 100
	MaxFileLength     = 32 << 20 // 32Mb
)
