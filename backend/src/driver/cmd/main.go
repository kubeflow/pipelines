// Copyright 2025 The Kubeflow Authors
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

// Binary driver-service is a standalone HTTP service that exposes the KFP
// driver as a centralized endpoint in the kubeflow namespace. Argo Workflows
// calls POST /api/v1/template.execute for each workflow driver node via an
// Argo HTTP template, eliminating per-workflow driver sidecar pods.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	driver "github.com/kubeflow/pipelines/backend/src/driver"
)

func main() {
	// glog writes to files by default; redirect to stderr so logs appear in
	// kubectl logs / container stdout.
	flag.Set("logtostderr", "true")
	flag.Parse()

	port := os.Getenv("KFP_DRIVER_PORT")
	if port == "" {
		port = "8080"
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/template.execute", driver.ExecutePlugin).Methods(http.MethodPost)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	}).Methods(http.MethodGet)

	addr := fmt.Sprintf(":%s", port)
	glog.Infof("KFP driver service listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		glog.Fatalf("driver service failed: %v", err)
	}
}
