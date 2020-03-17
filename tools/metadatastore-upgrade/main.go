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
	"path/filepath"
	"os"
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
	kubeconfigPath = flag.String("kubeconfig", "", "Absolute path to kubeconfig file.")
	imageTag = flag.String("image_tag", "", "Image tag to use for upgrade")
	deploymentNamespace = flag.String("namespace", "kubeflow", "Namespace where MetadataStore is deployed.")
)

const (
	ML_METADATA_DEPLOYMENT = "metadata-deployment"
	ML_METADATA_IMAGE = "gcr.io/tfx-oss-public/ml_metadata_store_server"
	UPGRADE_FLAG = "--enable_database_upgrade=true"
	MAX_WAIT_TIME = 30
)

func UpdateDeployment(deploymentsClient v1.DeploymentInterface, image string, containerArgs []string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Retrieve the latest version of Deployment before attempting update
			// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
			result, getErr := deploymentsClient.Get( ML_METADATA_DEPLOYMENT,  metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("Failed to get latest version of Deployment: %v", getErr)
			}

			result.Spec.Template.Spec.Containers[0].Image = image
			result.Spec.Template.Spec.Containers[0].Args = containerArgs
			_, updateErr := deploymentsClient.Update(result)
			if updateErr != nil {
				return updateErr
			}

			result, getErr = deploymentsClient.Get( ML_METADATA_DEPLOYMENT,  metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("Failed to get latest version of Deployment after update: %v", getErr)
			}

			waitTime := 0
			updateSuccessful := false
			for waitTime < MAX_WAIT_TIME && !updateSuccessful {
				if *result.Spec.Replicas != result.Status.ReadyReplicas {
					time.Sleep(time.Second)
					waitTime++
				} else {
					updateSuccessful = true
				}
		}

		if !updateSuccessful {
			return fmt.Errorf("Updated deployment failed to reach running state")
		}

		return nil
	})
}


func main() {
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
		log.Printf("Error reading kubeconfig file: %v", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Error setting up client auth: %v", err)
		os.Exit(1)
	}

	deploymentsClient := clientset.AppsV1().Deployments(*deploymentNamespace)

	oldDeployment, getErr := deploymentsClient.Get( ML_METADATA_DEPLOYMENT,  metav1.GetOptions{})
	if getErr != nil {
		log.Printf("Failed to get old Deployment: %v", getErr)
		os.Exit(1)
	}

	originalImage := oldDeployment.Spec.Template.Spec.Containers[0].Image
	originalContainerArgs := oldDeployment.Spec.Template.Spec.Containers[0].Args

	newImage := ML_METADATA_IMAGE + ":" + *imageTag
	log.Printf("Upgrading MetadataStore in Namespace: %s to Image: %s", *deploymentNamespace, newImage)
	updateErr := UpdateDeployment(deploymentsClient, newImage, append(originalContainerArgs, UPGRADE_FLAG))
	if updateErr == nil {
	  log.Printf("MetadataStore successfully upgraded to Image: %s", newImage)
	  log.Printf("Cleaning up Upgrade")
		upgradeCleanupErr := 	UpdateDeployment(deploymentsClient, newImage, originalContainerArgs)
		// In a highly unlikely scenario upgrade cleanup can fail.
		if upgradeCleanupErr != nil {
			log.Printf("Upgrade cleanup failed. Likely MetadataStore is in a functioning state. Please contact Kubeflow community for verification")
			os.Exit(1)
		}
	} else {
		log.Printf("MetadataStore Upgrade failed. Reverting back to Old Image")
		revertUpdateErr := UpdateDeployment(deploymentsClient, originalImage, originalContainerArgs)
		if revertUpdateErr != nil {
			log.Printf("Revert failed. Please contact Kubeflow community")
			os.Exit(1)
		} else {
			log.Printf("Revert succeeded. MetadataStore unchanged.")
			os.Exit(1)
		}
	}
}
