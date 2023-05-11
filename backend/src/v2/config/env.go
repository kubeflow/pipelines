// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metadata contains types to record/retrieve metadata stored in MLMD
// for individual pipeline steps.
package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/golang/glog"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	configMapName                = "kfp-launcher"
	defaultPipelineRoot          = "minio://mlpipeline/v2/artifacts"
	configKeyDefaultPipelineRoot = "defaultPipelineRoot"
)

// Config is the KFP runtime configuration.
type Config struct {
	data map[string]string
}

// FromConfigMap loads config from a kfp-launcher Kubernetes config map.
func FromConfigMap(ctx context.Context, clientSet kubernetes.Interface, namespace string) (*Config, error) {
	config, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if k8errors.IsNotFound(err) {
			glog.Infof("cannot find launcher configmap: name=%q namespace=%q, will use default config", configMapName, namespace)
			// LauncherConfig is optional, so ignore not found error.
			return nil, nil
		}
		return nil, err
	}
	return &Config{data: config.Data}, nil
}

// Config.DefaultPipelineRoot gets the configured default pipeline root.
func (c *Config) DefaultPipelineRoot() string {
	// The key defaultPipelineRoot is optional in launcher config.
	if c == nil || c.data[configKeyDefaultPipelineRoot] == "" {
		return defaultPipelineRoot
	}
	return c.data[configKeyDefaultPipelineRoot]
}

// InPodNamespace gets current namespace from inside a Kubernetes Pod.
func InPodNamespace() (string, error) {
	// The path is available in Pods.
	// https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api
	ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("failed to get namespace in Pod: %w", err)
	}
	return string(ns), nil
}

// InPodName gets the pod name from inside a Kubernetes Pod.
func InPodName() (string, error) {
	podName, err := ioutil.ReadFile("/etc/hostname")
	if err != nil {
		return "", fmt.Errorf("failed to get pod name in Pod: %w", err)
	}
	name := string(podName)
	return strings.TrimSuffix(name, "\n"), nil
}
