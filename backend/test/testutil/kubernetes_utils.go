// Copyright 2018-2023 The Kubeflow Authors
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

package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/onsi/gomega"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/exec" // Register exec credential plugin for OIDC authentication
)

func CreateK8sClient() (*kubernetes.Clientset, error) {
	restConfig, configErr := util.GetKubernetesConfig()
	if configErr != nil {
		return nil, configErr
	}
	k8sClient, clientErr := kubernetes.NewForConfig(restConfig)
	if clientErr != nil {
		return nil, clientErr
	}
	return k8sClient, nil
}

// ReadContainerLogs - Read pod logs from a specific container
func ReadContainerLogs(client *kubernetes.Clientset, namespace string, containerName string, follow *bool, sinceTime *time.Time, logLimit *int64) string {
	pod := GetPodContainingContainer(client, namespace, containerName)
	if pod != nil {
		return ReadPodLogs(client, namespace, pod.Name, follow, sinceTime, logLimit)
	} else {
		return fmt.Sprintf("Could not find pod containing container with name '%s'", containerName)
	}
}

// ReadPodLogs - Read pod logs from a specific names, with container name containing a substring and from a certain time period (default being from past 1 min)
func ReadPodLogs(client *kubernetes.Clientset, namespace string, podName string, follow *bool, sinceTime *time.Time, logLimit *int64) string {
	podFromPodName := GetPodContainingName(client, namespace, podName)
	podLogOptions := GetDefaultPodLogOptions()
	if logLimit != nil {
		podLogOptions.LimitBytes = logLimit
	}
	if follow != nil {
		podLogOptions.Follow = *follow
	}
	if sinceTime != nil {
		timeSince := metav1.NewTime(sinceTime.UTC())
		podLogOptions.SinceTime = &timeSince
	}
	buf := new(bytes.Buffer)
	if podFromPodName != nil {
		for _, container := range podFromPodName.Spec.Containers {
			podLogOptions.Container = container.Name
			podLogsRequest := client.CoreV1().Pods(namespace).GetLogs(podFromPodName.Name, podLogOptions)
			podLogs, err := podLogsRequest.Stream(context.Background()) // Pass a context for cancellation
			if err != nil {
				logger.Log("Failed to stream pod logs due to %v", err)
			}
			defer func(podLogs io.ReadCloser) {
				err = podLogs.Close()
				if err != nil {
					logger.Log("Failed to close pod log reader due to %v", err)
				}
			}(podLogs)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				logger.Log("Failed to add pod logs to buffer due to: %v", err)
			}
		}
	} else {
		logger.Log("No pod logs available for pod with name '%s'", podName)
	}
	return buf.String()
}

// GetDefaultPodLogOptions - Get default pod log options for the pod log reader API request
func GetDefaultPodLogOptions() *v1.PodLogOptions {
	logLimit := int64(50000000)
	sinceTime := metav1.NewTime(time.Now().Add(-1 * time.Minute).UTC())
	return &v1.PodLogOptions{
		Previous:   false,
		SinceTime:  &sinceTime,
		Timestamps: true,
		LimitBytes: &logLimit,
		Follow:     false,
	}
}

// GetPodContainingName - Get the name of the pod with name containing substring
func GetPodContainingName(client *kubernetes.Clientset, namespace, podName string) *v1.Pod {
	pods, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Log("Failed to list pods due to: %v", err)
	}
	for _, pod := range pods.Items {
		podNameSplit := strings.Split(pod.Name, "-")
		expectedPodNameSplit := strings.Split(podName, "-")
		contains := true
		for _, name := range expectedPodNameSplit {
			if !slices.Contains(podNameSplit, name) {
				contains = false
			}
		}
		if contains {
			return &pod
		}
	}
	return nil
}

// GetPodContainingContainer - Get the name of the pod with container name containing substring
func GetPodContainingContainer(client *kubernetes.Clientset, namespace, containerName string) *v1.Pod {
	pods, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Log("Failed to list pods due to: %v", err)
	}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Name, containerName) {
				return &pod
			}
		}
	}
	return nil
}

func CreateUserToken(client *kubernetes.Clientset, namespace, serviceAccountName string) string {
	// Define TokenRequest
	tokenReq := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{"pipelines.kubeflow.org"},                             // Token for Kubernetes API server
			ExpirationSeconds: func(i int64) *int64 { return &i }(int64(time.Hour.Seconds())), // 1-hour expiration
		},
	}

	// Create the token
	tokenResponse, err := client.CoreV1().ServiceAccounts(namespace).CreateToken(context.TODO(), serviceAccountName, tokenReq, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), fmt.Sprintf("Failed to create service account token for '%s' service account under '%s' namespace", serviceAccountName, namespace))
	if tokenResponse != nil {
		return tokenResponse.Status.Token
	}
	return ""
}

// CreateSecret - Create K8s secret in the provided namespace
func CreateSecret(client *kubernetes.Clientset, namespace string, secret *v1.Secret) {
	_, createErr := client.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if createErr == nil {
		logger.Log("%s created", secret.Name)
	} else {
		logger.Log("Looks like %s already exists, because creation failed due to %s", secret.Name, createErr.Error())
		_, getErr := client.CoreV1().Secrets(namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
		gomega.Expect(getErr).ToNot(gomega.HaveOccurred(), "Failed to get secret '%s'")
	}
}
