// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type headers []string

func (values *headers) String() string         { return strings.Join(*values, ",") }
func (values *headers) Set(value string) error { *values = append(*values, value); return nil }

func run() error {
	var address, dsn, namespace, deployment, recurringDeployment string
	var useTLS, drainCheck, gateOnly bool
	var extraHeaders headers
	flag.StringVar(&address, "mlmd-address", "", "MLMD gRPC host:port")
	flag.StringVar(&dsn, "mysql-dsn", "", "MySQL driver DSN for the KFP database")
	flag.StringVar(&namespace, "namespace", "kubeflow", "namespace of the API Deployment")
	flag.StringVar(&deployment, "deployment", "ml-pipeline", "API Deployment to fence")
	flag.StringVar(&recurringDeployment, "recurring-deployment", "ml-pipeline-scheduledworkflow", "recurring-run controller Deployment to pause")
	flag.BoolVar(&useTLS, "mlmd-tls", false, "use TLS for the MLMD connection")
	flag.BoolVar(&drainCheck, "drain-check", false, "verify terminal KFP runs and Workflow reports before fencing old replicas")
	flag.BoolVar(&gateOnly, "gate-only", false, "verify the full writer fence without migrating")
	flag.Var(&extraHeaders, "mlmd-header", "gRPC header as key:value (repeatable)")
	flag.Parse()
	if dsn == "" {
		dsn = os.Getenv("KFP_MLMD_MYSQL_DSN")
	}
	if dsn == "" || (!drainCheck && !gateOnly && address == "") || (drainCheck && gateOnly) {
		return fmt.Errorf("provide a MySQL DSN and, for migration, --mlmd-address; choose at most one check mode")
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open KFP MySQL database: %w", err)
	}
	defer database.Close()
	kubeConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return fmt.Errorf("load Kubernetes config for writer fence: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	workflowClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	gate := mlmd.DrainGate{Database: database, Kubernetes: kubeClient, Workflows: workflowClient, Namespace: namespace, Deployment: deployment, RecurringDeployment: recurringDeployment}
	if drainCheck {
		if err := gate.CheckDrained(ctx); err != nil {
			return fmt.Errorf("drain check failed: %w", err)
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "drained"})
	}
	if err := gate.Check(ctx); err != nil {
		return fmt.Errorf("migration refused: %w", err)
	}
	if gateOnly {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "fenced"})
	}
	var transport = insecure.NewCredentials()
	if useTLS {
		transport = credentials.NewTLS(nil)
	}
	connection, err := grpc.NewClient(address, grpc.WithTransportCredentials(transport))
	if err != nil {
		return fmt.Errorf("connect to MLMD: %w", err)
	}
	defer connection.Close()
	metadataValues := metadata.MD{}
	for _, value := range extraHeaders {
		key, content, found := strings.Cut(value, ":")
		if !found || key == "" {
			return fmt.Errorf("invalid --mlmd-header %q: expected key:value", value)
		}
		metadataValues.Append(key, strings.TrimSpace(content))
	}
	ctx = metadata.NewOutgoingContext(ctx, metadataValues)
	report, err := mlmd.Migrate(ctx, pb.NewMetadataStoreServiceClient(connection), database)
	if err != nil {
		return fmt.Errorf("migration incomplete: %w", err)
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Status string `json:"status"`
		mlmd.Report
	}{Status: "complete", Report: report})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
