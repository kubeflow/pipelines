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
	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	k8score "k8s.io/api/core/v1"
	"os"
	"strconv"
	"strings"
)

const (
	MLPipelineTLSEnabledEnvVar  = "ML_PIPELINE_TLS_ENABLED"
	DefaultMLPipelineTLSEnabled = false
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

// ConfigureCABundle adds CABundle environment variables and volume mounts
// if CA Bundle env vars are specified.
func ConfigureCABundle(tmpl *wfapi.Template) {
	caBundleDir := common.CABundleDir
	caBundleSecretName := os.Getenv(common.CaBundleSecretName)
	if caBundleSecretName != "" {
		var certDirectories = []string{
			caBundleDir,
			"/etc/ssl/certs",
			"/etc/pki/tls/certs",
		}
		// OpenSSL default cert file env variable.
		// Similar to AWS_CA_BUNDLE, the SSL_CERT_DIR equivalent for paths had unyielding
		// results, even after rehashing.
		// https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_set_default_verify_paths.html
		tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
			Name:  "SSL_CERT_FILE",
			Value: common.TLSCertCAPath,
		})
		sslCertDir := strings.Join(certDirectories, ":")
		tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
			Name:  "SSL_CERT_DIR",
			Value: sslCertDir,
		})
		volume := k8score.Volume{
			Name: "ca-secret",
			VolumeSource: k8score.VolumeSource{
				Secret: &k8score.SecretVolumeSource{
					SecretName: caBundleSecretName,
				},
			},
		}

		tmpl.Volumes = append(tmpl.Volumes, volume)

		volumeMount := k8score.VolumeMount{
			Name:      "ca-secret",
			MountPath: caBundleDir,
		}

		tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, volumeMount)

	}
}

// addExitTask adds an exit lifecycle hook to a task if exitTemplate is not empty.
func addExitTask(task *wfapi.DAGTask, exitTemplate string, parentDagID string) {
	if exitTemplate == "" {
		return
	}

	task.Hooks = wfapi.LifecycleHooks{
		wfapi.ExitLifecycleEvent: wfapi.LifecycleHook{
			Template: exitTemplate,
			Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{
				{Name: paramParentDagID, Value: wfapi.AnyStringPtr(parentDagID)},
			}},
		},
	}
}
