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

// Package plugins configs common across all plugins, across the API server and driver/launcher.
package plugins

import (
	"encoding/json"
	"fmt"
)

// Auth type constants.
const (
	AuthTypeKubernetes = "kubernetes"
	AuthTypeBearer     = "bearer"
	AuthTypeBasicAuth  = "basic-auth"
	AuthTypeNone       = "none"
)

// TLSConfig holds TLS settings for a plugin endpoint.
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty" mapstructure:"insecureSkipVerify"`
	CABundlePath       string `json:"caBundlePath,omitempty" mapstructure:"caBundlePath"`
}

// CredentialSecretRef identifies the Secret keys used for MLflow auth.
type CredentialSecretRef struct {
	TokenKey    string `json:"tokenKey,omitempty"`
	UsernameKey string `json:"usernameKey,omitempty"`
	PasswordKey string `json:"passwordKey,omitempty"`
}

func (r *CredentialSecretRef) UnmarshalJSON(data []byte) error {
	type credentialSecretRef CredentialSecretRef
	var raw struct {
		credentialSecretRef
		Name string `json:"name,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Name != "" {
		return fmt.Errorf("credentialSecretRef.name is not supported; Secret name is fixed to %q", "kfp-mlflow-credentials")
	}
	*r = CredentialSecretRef(raw.credentialSecretRef)
	return nil
}
