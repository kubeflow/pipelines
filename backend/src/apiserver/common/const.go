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

const DefaultMetadataTLSEnabled = false

const (
	DefaultPipelineRunnerServiceAccount = "pipeline-runner"
	HasDefaultBucketEnvVar              = "HAS_DEFAULT_BUCKET"
	DefaultBucketNameEnvVar             = "BUCKET_NAME"
	ProjectIDEnvVar                     = "PROJECT_ID"
)

const (
	MaxFileNameLength = 100
	MaxFileLength     = 32 << 20 // 32Mb
)

const (
	CustomCaCertPath = "/kfp/certs/ca.crt"
	CABundleDir      = "/kfp/certs"
)

const (
	DefaultPodNamespace string = "kubeflow"
)

const (
	DefaultMLPipelineServiceName string = "ml-pipeline"
	DefaultMetadataServiceName   string = "metadata-grpc-service"
	DefaultClusterDomain         string = "cluster.local"
)

// ClearTagsMetadataKey is the gRPC metadata key set by the HTTP middleware
// when the client sends an empty tags map ("tags":{}) to signal that all
// tags should be removed. Protobuf binary encoding cannot distinguish an
// empty map from nil, so this header preserves the intent across the
// HTTP→gRPC proxy roundtrip.
const ClearTagsMetadataKey = "x-clear-tags"
// Pod lifecycle failure timeout configuration.
// These environment variables allow operators to configure how long KFP
// waits before marking a run as FAILED when a pod is stuck in a
// lifecycle failure state. Each category maps to the three failure types
// described in issue #12843.
const (
	// Provisioning timeout covers failures like ImagePullBackOff,
	// ErrImagePull and Unschedulable where the pod cannot be scheduled
	// or its container image cannot be pulled.
	PodLifecycleProvisioningTimeoutEnvVar = "KFP_POD_LIFECYCLE_PROVISIONING_TIMEOUT_SECONDS"

	// Runtime timeout covers failures like OOMKilled and CrashLoopBackOff
	// where the pod starts but fails during execution.
	PodLifecycleRuntimeTimeoutEnvVar = "KFP_POD_LIFECYCLE_RUNTIME_TIMEOUT_SECONDS"

	// Node timeout covers infrastructure failures like NodeLost,
	// Preempted and Evicted where the underlying node fails.
	PodLifecycleNodeTimeoutEnvVar = "KFP_POD_LIFECYCLE_NODE_TIMEOUT_SECONDS"

	// Default timeout of 1 hour for all pod lifecycle failure categories
	// if no environment variable is set.
	DefaultPodLifecycleTimeoutSeconds = 3600
)
