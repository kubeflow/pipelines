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

package clientmanager

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
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

func migrateMLMDOnStartup(ctx context.Context, db *sql.DB) error {
	required, err := mlmd.NeedsMigration(ctx, db)
	if err != nil {
		return err
	}
	transferPending, err := needsTransferIdentityUpgrade(ctx, db)
	if err != nil {
		return err
	}
	if !required && !transferPending {
		return nil
	}
	host, port := os.Getenv("METADATA_GRPC_SERVICE_HOST"), os.Getenv("METADATA_GRPC_SERVICE_PORT")
	if required && (host == "" || port == "") {
		return fmt.Errorf("legacy MLMD schema requires METADATA_GRPC_SERVICE_HOST and METADATA_GRPC_SERVICE_PORT; configure them for the upgrade boot")
	}
	podName, namespace := os.Getenv("POD_NAME"), common.GetPodNamespace()
	if podName == "" || namespace == "" {
		return fmt.Errorf("MLMD startup migration requires POD_NAME and pod namespace to verify the old API writer fence")
	}
	kubeConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return fmt.Errorf("load Kubernetes config for MLMD drain gate: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	workflowClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	if err := (mlmd.DrainGate{Database: db, Kubernetes: kubeClient, Workflows: workflowClient, Namespace: namespace, Deployment: "ml-pipeline", PodName: podName}).Check(ctx); err != nil {
		return fmt.Errorf("MLMD startup migration refused: %w", err)
	}
	if required {
		transport := insecure.NewCredentials()
		if strings.EqualFold(os.Getenv("MLMD_MIGRATION_TLS"), "true") {
			transport = credentials.NewTLS(nil)
		}
		connection, err := grpc.NewClient(net.JoinHostPort(host, port), grpc.WithTransportCredentials(transport))
		if err != nil {
			return fmt.Errorf("connect to MLMD: %w", err)
		}
		defer connection.Close()
		if raw := os.Getenv("MLMD_MIGRATION_HEADERS_JSON"); raw != "" {
			var headers map[string]string
			if err := json.Unmarshal([]byte(raw), &headers); err != nil {
				return fmt.Errorf("parse MLMD_MIGRATION_HEADERS_JSON: %w", err)
			}
			ctx = metadata.NewOutgoingContext(ctx, metadata.New(headers))
		}
		glog.Info("MLMD schema detected; running startup migration under the maintenance writer fence")
		report, err := mlmd.Migrate(ctx, pb.NewMetadataStoreServiceClient(connection), db)
		if err != nil {
			return fmt.Errorf("MLMD startup migration incomplete: %w", err)
		}
		stillRequired, err := mlmd.NeedsMigration(ctx, db)
		if err != nil {
			return fmt.Errorf("verify MLMD migration completion: %w", err)
		}
		if stillRequired {
			return fmt.Errorf("MLMD migration completion predicate did not pass; inspect migration tables before restarting")
		}
		for _, skipped := range report.Skipped {
			glog.Warningf("MLMD migration skipped %s %d: %s", skipped.Kind, skipped.ID, skipped.Reason)
		}
		for _, warning := range report.Warnings {
			glog.Warningf("MLMD migration: %s", warning)
		}
		glog.Infof("MLMD startup migration complete: %d tasks, %d artifacts, %d links, %d skipped", report.Tasks, report.Artifacts, report.Links, len(report.Skipped))
	}
	if err := reconcileLegacyTransferIdentity(ctx, db); err != nil {
		return fmt.Errorf("reconcile released transfer identity before AutoMigrate: %w", err)
	}
	if transferPending {
		glog.Info("Released transfer identity reconciled before native startup")
	}
	return nil
}
