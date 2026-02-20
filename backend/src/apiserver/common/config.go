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

import (
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

const (
	MultiUserMode                           string = "MULTIUSER"
	MultiUserModeSharedReadAccess           string = "MULTIUSER_SHARED_READ"
	PodNamespace                            string = "POD_NAMESPACE"
	CacheEnabled                            string = "CacheEnabled"
	DefaultPipelineRunnerServiceAccountFlag string = "DEFAULTPIPELINERUNNERSERVICEACCOUNT"
	KubeflowUserIDHeader                    string = "KUBEFLOW_USERID_HEADER"
	KubeflowUserIDPrefix                    string = "KUBEFLOW_USERID_PREFIX"
	UpdatePipelineVersionByDefault          string = "AUTO_UPDATE_PIPELINE_DEFAULT_VERSION"
	TokenReviewAudience                     string = "TOKEN_REVIEW_AUDIENCE"
	MetadataTLSEnabled                      string = "METADATA_TLS_ENABLED"
	CaBundleSecretName                      string = "CABUNDLE_SECRET_NAME"
	CaBundleConfigMapName                   string = "CABUNDLE_CONFIGMAP_NAME"
	CaBundleKeyName                         string = "CABUNDLE_KEY_NAME"
	RequireNamespaceForPipelines            string = "REQUIRE_NAMESPACE_FOR_PIPELINES"
	CompiledPipelineSpecPatch               string = "COMPILED_PIPELINE_SPEC_PATCH"
	MLPipelineServiceName                   string = "ML_PIPELINE_SERVICE_NAME"
	MetadataServiceName                     string = "METADATA_SERVICE_NAME"
	ClusterDomain                           string = "CLUSTER_DOMAIN"
	DefaultSecurityContextRunAsUser         string = "DEFAULT_SECURITY_CONTEXT_RUN_AS_USER"
	DefaultSecurityContextRunAsGroup        string = "DEFAULT_SECURITY_CONTEXT_RUN_AS_GROUP"
	DefaultSecurityContextRunAsNonRoot      string = "DEFAULT_SECURITY_CONTEXT_RUN_AS_NON_ROOT"
)

func IsPipelineVersionUpdatedByDefault() bool {
	return GetBoolConfigWithDefault(UpdatePipelineVersionByDefault, true)
}

func IsNamespaceRequiredForPipelines() bool {
	return GetBoolConfigWithDefault(RequireNamespaceForPipelines, false)
}

func GetStringConfig(configName string) string {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetString(configName)
}

func GetStringConfigWithDefault(configName, value string) string {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetString(configName)
}

func GetMapConfig(configName string) map[string]string {
	if !viper.IsSet(configName) {
		glog.Infof("Config %s not specified, skipping", configName)
		return nil
	}
	return viper.GetStringMapString(configName)
}

func GetBoolConfigWithDefault(configName string, value bool) bool {
	if !viper.IsSet(configName) {
		return value
	}
	value, err := strconv.ParseBool(viper.GetString(configName))
	if err != nil {
		glog.Fatalf("Failed converting string to bool %s", viper.GetString(configName))
	}
	return value
}

func GetFloat64ConfigWithDefault(configName string, value float64) float64 {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetFloat64(configName)
}

func GetIntConfigWithDefault(configName string, value int) int {
	if !viper.IsSet(configName) {
		return value
	}
	return viper.GetInt(configName)
}

func GetDurationConfig(configName string) time.Duration {
	if !viper.IsSet(configName) {
		glog.Fatalf("Please specify flag %s", configName)
	}
	return viper.GetDuration(configName)
}

func IsMultiUserMode() bool {
	return GetBoolConfigWithDefault(MultiUserMode, false)
}

func IsMultiUserSharedReadMode() bool {
	return GetBoolConfigWithDefault(MultiUserModeSharedReadAccess, false)
}

func GetPodNamespace() string {
	return GetStringConfigWithDefault(PodNamespace, DefaultPodNamespace)
}

func GetMLPipelineServiceName() string {
	return GetStringConfigWithDefault(MLPipelineServiceName, DefaultMLPipelineServiceName)
}

func GetMetadataServiceName() string {
	return GetStringConfigWithDefault(MetadataServiceName, DefaultMetadataServiceName)
}

func GetClusterDomain() string {
	return GetStringConfigWithDefault(ClusterDomain, DefaultClusterDomain)
}

func GetBoolFromStringWithDefault(value string, defaultValue bool) bool {
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolVal
}

func IsCacheEnabled() string {
	return GetStringConfigWithDefault(CacheEnabled, "true")
}

func GetKubeflowUserIDHeader() string {
	return GetStringConfigWithDefault(KubeflowUserIDHeader, GoogleIAPUserIdentityHeader)
}

func GetKubeflowUserIDPrefix() string {
	return GetStringConfigWithDefault(KubeflowUserIDPrefix, GoogleIAPUserIdentityPrefix)
}

func GetTokenReviewAudience() string {
	return GetStringConfigWithDefault(TokenReviewAudience, DefaultTokenReviewAudience)
}

func GetMetadataTLSEnabled() bool {
	return GetBoolConfigWithDefault(MetadataTLSEnabled, DefaultMetadataTLSEnabled)
}

func GetCaBundleSecretName() string {
	return GetStringConfigWithDefault(CaBundleSecretName, "")
}

func GetCABundleKey() string {
	return GetStringConfigWithDefault(CaBundleKeyName, "")
}

func GetCaBundleConfigMapName() string {
	return GetStringConfigWithDefault(CaBundleConfigMapName, "")
}

func GetCompiledPipelineSpecPatch() string {
	return GetStringConfigWithDefault(CompiledPipelineSpecPatch, "{}")
}

func GetDefaultSecurityContextRunAsUser() string {
	return GetStringConfigWithDefault(DefaultSecurityContextRunAsUser, "")
}

func GetDefaultSecurityContextRunAsGroup() string {
	return GetStringConfigWithDefault(DefaultSecurityContextRunAsGroup, "")
}

func GetDefaultSecurityContextRunAsNonRoot() string {
	return GetStringConfigWithDefault(DefaultSecurityContextRunAsNonRoot, "")
}
