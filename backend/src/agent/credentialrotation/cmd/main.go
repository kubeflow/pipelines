// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	rotation "github.com/kubeflow/pipelines/backend/src/agent/credentialrotation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() { os.Exit(run()) }

func run() int {
	mode := flag.String("mode", "migrate", "bootstrap prepares credentials and a Job; migrate completes and exits")
	namespace := flag.String("namespace", os.Getenv("POD_NAMESPACE"), "installation namespace")
	image := flag.String("job-image", os.Getenv("MIGRATION_IMAGE"), "image containing this binary for the migration Job")
	flag.Parse()
	if *namespace == "" || (*mode != "bootstrap" && *mode != "migrate") {
		log.Print("Set POD_NAMESPACE and a valid mode")
		return 2
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Cannot initialize Kubernetes credentials (%T)", err)
		return 1
	}
	config.Timeout = 20 * time.Second
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Cannot initialize Kubernetes client (%T)", err)
		return 1
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if *mode == "bootstrap" && *image == "" {
		podName := os.Getenv("POD_NAME")
		if podName == "" {
			log.Print("Bootstrap requires POD_NAME to resolve its image")
			return 2
		}
		pod, err := client.CoreV1().Pods(*namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Cannot read bootstrap Pod (%T)", err)
			return 1
		}
		for _, container := range pod.Spec.InitContainers {
			if container.Name == "prepare-storage-credentials" {
				*image = container.Image
			}
		}
		if *image == "" {
			log.Print("Bootstrap image is missing from Pod specification")
			return 2
		}
	}
	migration := rotation.New(&rotation.Client{Clientset: client, Namespace: *namespace}, func(pair rotation.Credentials) (rotation.Storage, error) {
		return rotation.NewRemoteStorage(pair, "http://seaweedfs:8333", "http://seaweedfs:8111")
	})
	defer migration.Close()
	health := &rotation.Health{}
	if *mode == "migrate" {
		mux := http.NewServeMux()
		mux.Handle("/readyz", health)
		server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		defer server.Close()
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Readiness server failed (%T)", err)
				cancel()
			}
		}()
	}
	for {
		if *mode == "bootstrap" {
			state, prepareErr := migration.Prepare(ctx)
			err = prepareErr
			if err == nil {
				err = rotation.EnsureMigrationJob(ctx, client, *namespace, *image, int64(os.Getuid()), state)
			}
		} else {
			err = migration.Reconcile(ctx)
		}
		health.Observe(err)
		if err == nil {
			log.Printf("Credential %s completed", *mode)
			return 0
		}
		if rotation.IsWaiting(err) {
			log.Print(err)
		} else {
			// Never print API/SDK error bodies: they can contain credential data.
			log.Printf("Credential %s failed during %s (%T); inspect configuration, RBAC and storage health; retrying", *mode, migration.Stage(), err)
		}
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 1
		case <-timer.C:
		}
	}
}
