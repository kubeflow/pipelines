// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"fmt"
	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	k8score "k8s.io/api/core/v1"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultMLPipelineServiceHost     = "ml-pipeline.kubeflow.svc.cluster.local"
	DefaultMLPipelineServicePortGRPC = "8887"
	MLPipelineServiceHostEnvVar      = "ML_PIPELINE_SERVICE_HOST"
	MLPipelineServicePortGRPCEnvVar  = "ML_PIPELINE_SERVICE_PORT_GRPC"
	MLPipelineTLSEnabledEnvVar       = "ML_PIPELINE_TLS_ENABLED"
	DefaultMLPipelineTLSEnabled      = false
)

// env vars in metadata-grpc-configmap is defined in component package
var metadataConfigIsOptional bool = true
var metadataEnvFrom = k8score.EnvFromSource{
	ConfigMapRef: &k8score.ConfigMapEnvSource{
		LocalObjectReference: k8score.LocalObjectReference{
			Name: "metadata-grpc-configmap",
		},
		Optional: &metadataConfigIsOptional,
	},
}

var commonEnvs = []k8score.EnvVar{{
	Name: "KFP_POD_NAME",
	ValueFrom: &k8score.EnvVarSource{
		FieldRef: &k8score.ObjectFieldSelector{
			FieldPath: "metadata.name",
		},
	},
}, {
	Name: "KFP_POD_UID",
	ValueFrom: &k8score.EnvVarSource{
		FieldRef: &k8score.ObjectFieldSelector{
			FieldPath: "metadata.uid",
		},
	},
}}

var MLPipelineServiceEnv = []k8score.EnvVar{{
	Name:  "ML_PIPELINE_SERVICE_HOST",
	Value: GetMLPipelineServiceHost(),
}, {
	Name:  "ML_PIPELINE_SERVICE_PORT_GRPC",
	Value: GetMLPipelineServicePortGRPC(),
}}

func GetMLPipelineServiceTLSEnabled() (bool, error) {
	mlPipelineServiceTLSEnabledStr := os.Getenv(MLPipelineTLSEnabledEnvVar)
	if mlPipelineServiceTLSEnabledStr == "" {
		return DefaultMLPipelineTLSEnabled, nil
	}
	mlPipelineServiceTLSEnabled, err := strconv.ParseBool(os.Getenv(MLPipelineTLSEnabledEnvVar))
	if err != nil {
		return false, err
	}
	return mlPipelineServiceTLSEnabled, nil
}

func GetMLPipelineServiceHost() string {
	mlPipelineServiceHost := os.Getenv(MLPipelineServiceHostEnvVar)
	if mlPipelineServiceHost == "" {
		return DefaultMLPipelineServiceHost
	}
	return mlPipelineServiceHost
}

func GetMLPipelineServicePortGRPC() string {
	mlPipelineServicePortGRPC := os.Getenv(MLPipelineServicePortGRPCEnvVar)
	if mlPipelineServicePortGRPC == "" {
		return DefaultMLPipelineServicePortGRPC
	}
	return mlPipelineServicePortGRPC
}

// ConfigureCABundle adds CABundle environment variables and volume mounts
// if CA Bundle env vars are specified.
func ConfigureCABundle(tmpl *wfapi.Template) {
	caBundleCfgMapName := os.Getenv("ARTIFACT_COPY_STEP_CABUNDLE_CONFIGMAP_NAME")
	caBundleCfgMapKey := os.Getenv("ARTIFACT_COPY_STEP_CABUNDLE_CONFIGMAP_KEY")
	caBundleMountPath := os.Getenv("ARTIFACT_COPY_STEP_CABUNDLE_MOUNTPATH")
	if caBundleCfgMapName != "" && caBundleCfgMapKey != "" {
		caFile := fmt.Sprintf("%s/%s", caBundleMountPath, caBundleCfgMapKey)
		var certDirectories = []string{
			caBundleMountPath,
			"/etc/ssl/certs",
			"/etc/pki/tls/certs",
		}
		// Add to REQUESTS_CA_BUNDLE for python request library.
		// As many python web based libraries utilize this, we add it here so the user
		// does not have to manually include this in the user pipeline.
		// Note: for packages like Boto3, even though it is documented to use AWS_CA_BUNDLE,
		// we found the python boto3 client only works if we include REQUESTS_CA_BUNDLE.
		// https://requests.readthedocs.io/en/latest/user/advanced/#ssl-cert-verification
		// https://github.com/aws/aws-cli/issues/3425
		tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
			Name:  "REQUESTS_CA_BUNDLE",
			Value: caFile,
		})
		// For AWS utilities like cli, and packages.
		tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
			Name:  "AWS_CA_BUNDLE",
			Value: caFile,
		})
		// OpenSSL default cert file env variable.
		// Similar to AWS_CA_BUNDLE, the SSL_CERT_DIR equivalent for paths had unyielding
		// results, even after rehashing.
		// https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_set_default_verify_paths.html
		tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
			Name:  "SSL_CERT_FILE",
			Value: caFile,
		})
		sslCertDir := strings.Join(certDirectories, ":")
		tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
			Name:  "SSL_CERT_DIR",
			Value: sslCertDir,
		})
		volume := k8score.Volume{
			Name: volumeNameCABUndle,
			VolumeSource: k8score.VolumeSource{
				ConfigMap: &k8score.ConfigMapVolumeSource{
					LocalObjectReference: k8score.LocalObjectReference{
						Name: caBundleCfgMapName,
					},
				},
			},
		}

		tmpl.Volumes = append(tmpl.Volumes, volume)

		volumeMount := k8score.VolumeMount{
			Name:      volumeNameCABUndle,
			MountPath: caFile,
			SubPath:   caBundleCfgMapKey,
		}

		tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, volumeMount)

	}
}
