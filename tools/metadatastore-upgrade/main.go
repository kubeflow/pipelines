// Copyright 2020 Google LLC
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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	kubeconfigPath      = flag.String("kubeconfig", "", "Absolute path to kubeconfig file.")
	imageTag            = flag.String("new_image_tag", "", "Image tag of a released ML Metadata store server image in repository gcr.io/tfx-oss-public/ml_metadata_store_server.")
	deploymentNamespace = flag.String("namespace", "kubeflow", "Namespace of the Metadata deployment in the cluster.")
)

const (
	mlMetadataDeployment = "metadata-deployment"
	mlMetadataImage      = "gcr.io/tfx-oss-public/ml_metadata_store_server"
	upgradeFlag          = "--enable_database_upgrade=true"
	maxWaitTime          = 30
)

func updateDeployment(deploymentsClient v1.DeploymentInterface, image string, containerArgs []string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		result, err := deploymentsClient.Get(mlMetadataDeployment, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get latest version of Deployment: %v", err)
		}

		result.Spec.Template.Spec.Containers[0].Image = image
		result.Spec.Template.Spec.Containers[0].Args = containerArgs
		_, err = deploymentsClient.Update(result)
		if err != nil {
			return err
		}

		result, err = deploymentsClient.Get(mlMetadataDeployment, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get latest version of Deployment after update: %v", err)
		}

		waitTime := 0
		updateSuccessful := false
		for waitTime < maxWaitTime && !updateSuccessful {
			if *result.Spec.Replicas != result.Status.ReadyReplicas {
				time.Sleep(time.Second)
				waitTime++
			} else {
				updateSuccessful = true
			}
		}

		if !updateSuccessful {
			return fmt.Errorf("updated deployment failed to reach running state")
		}

		return nil
	})
}

func main() {
	var err error
	flag.Parse()
	log.SetOutput(os.Stdout)

	if *imageTag == "" {
		log.Printf("Missing image_tag flag")
		flag.Usage()
		os.Exit(1)
	}

	if *kubeconfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			*kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfigPath)
	if err != nil {
		log.Fatalf("Error reading kubeconfig file: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error setting up client auth: %v", err)
	}

	deploymentsClient := clientset.AppsV1().Deployments(*deploymentNamespace)

	originalDeployment, err := deploymentsClient.Get(mlMetadataDeployment, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to get old Deployment: %v", err)
	}

	originalImage := originalDeployment.Spec.Template.Spec.Containers[0].Image
	originalContainerArgs := originalDeployment.Spec.Template.Spec.Containers[0].Args

	newImage := mlMetadataImage + ":" + *imageTag
	log.Printf("Upgrading MetadataStore in Namespace: %s from Image: %s to Image: %s", *deploymentNamespace, originalImage, newImage)
	err = updateDeployment(deploymentsClient, newImage, append(originalContainerArgs, upgradeFlag))
	if err == nil {
		log.Printf("MetadataStore successfully upgraded to Image: %s", newImage)
		log.Printf("Cleaning up Upgrade")
		err = updateDeployment(deploymentsClient, newImage, originalContainerArgs)
		// In a highly unlikely scenario upgrade cleanup can fail.
		if err != nil {
			log.Fatalf("Upgrade cleanup failed: %v. \nLikely MetadataStore is in a functioning state.", err)
		}
	} else {
		log.Printf("MetadataStore Upgrade failed: %v. \nReverting back to Old Image", err)
		err = updateDeployment(deploymentsClient, originalImage, originalContainerArgs)
		if err != nil {
			log.Fatalf("Revert failed: %v", err)
		} else {
			log.Fatalf("Upgrade effort failed. MetadataStore unchanged.")
		}
	}
}
