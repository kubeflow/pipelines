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
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func migrateMLMDOnStartup(ctx context.Context, db *sql.DB, dialect string, allowCutover bool) error {
	state, err := mlmd.ClassifyStartup(ctx, db, dialect)
	if err != nil {
		return err
	}
	if dialect != "mysql" {
		return nil
	}
	migrationPending := state == mlmd.MigrationRequired || state == mlmd.CleanupPending
	transferPending, err := needsTransferIdentityUpgrade(ctx, db)
	if err != nil {
		return err
	}
	if !migrationPending && !transferPending {
		return nil
	}
	if !allowCutover {
		return fmt.Errorf("%s", mlmd.CutoverHoldMessage)
	}
	host, port := os.Getenv("METADATA_GRPC_SERVICE_HOST"), os.Getenv("METADATA_GRPC_SERVICE_PORT")
	if state == mlmd.MigrationRequired && (host == "" || port == "") {
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
	kubeConfig.Timeout = 30 * time.Second
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	workflowClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	sourceSingleUser := strings.EqualFold(os.Getenv("MLMD_MIGRATION_SOURCE_SINGLE_USER"), "true")
	if sourceSingleUser && common.IsMultiUserMode() {
		return fmt.Errorf("historical single-user fallback requires a single-user target API server")
	}
	gate := mlmd.DrainGate{SourceSingleUser: sourceSingleUser, Database: db, Kubernetes: kubeClient, Workflows: workflowClient, Namespace: namespace, Deployment: "ml-pipeline", PodName: podName}
	if err := gate.Check(ctx); err != nil {
		return fmt.Errorf("MLMD startup migration refused: %w", err)
	}
	var source pb.MetadataStoreServiceClient
	var budget mlmd.DiskBudget
	if state == mlmd.MigrationRequired {
		for name, value := range map[string]*uint64{
			"MLMD_MIGRATION_AVAILABLE_DISK_BYTES": &budget.AvailableBytes,
			"MLMD_MIGRATION_DATABASE_BYTES":       &budget.MLMDDatabaseBytes,
		} {
			raw := os.Getenv(name)
			if raw == "" {
				continue
			}
			bytes, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				return fmt.Errorf("%s must be a byte count", name)
			}
			*value = bytes
		}
		connection, err := mlmd.DialSource(net.JoinHostPort(host, port), strings.EqualFold(os.Getenv("MLMD_MIGRATION_TLS"), "true"))
		if err != nil {
			return err
		}
		defer connection.Close()
		if raw := os.Getenv("MLMD_MIGRATION_HEADERS_JSON"); raw != "" {
			var headers map[string]string
			if err := json.Unmarshal([]byte(raw), &headers); err != nil {
				return fmt.Errorf("parse MLMD_MIGRATION_HEADERS_JSON: %w", err)
			}
			ctx = metadata.NewOutgoingContext(ctx, metadata.New(headers))
		}
		source = pb.NewMetadataStoreServiceClient(connection)
	}
	if migrationPending {
		glog.Info("MLMD schema detected; running startup migration under the maintenance writer fence")
		report, err := mlmd.MigrateFenced(ctx, source, db, budget, gate)
		for _, skipped := range report.Skipped {
			glog.Warningf("MLMD migration skipped %s %d (%s): %s", skipped.Kind, skipped.ID, skipped.Source, skipped.Reason)
		}
		for _, warning := range report.Warnings {
			glog.Warningf("MLMD migration: %s", warning)
		}
		if err != nil {
			return fmt.Errorf("MLMD startup migration incomplete: %w", err)
		}
		state, err := mlmd.ClassifyStartup(ctx, db, dialect)
		if err != nil {
			return fmt.Errorf("verify MLMD migration completion: %w", err)
		}
		if state != mlmd.Native {
			return fmt.Errorf("MLMD migration completion predicate did not pass; inspect migration tables before restarting")
		}
		glog.Infof("MLMD startup migration complete: %d tasks, %d artifacts, %d links; copied=%v skipped=%v skip_reasons=%v",
			report.Tasks, report.Artifacts, report.Links, report.MigratedByKind, report.SkippedByKind, report.SkippedByReason)
		transferPending, err = needsTransferIdentityUpgrade(ctx, db)
		if err != nil {
			return err
		}
	}
	if transferPending {
		if err := reconcileLegacyTransferIdentity(ctx, db, &gate); err != nil {
			return fmt.Errorf("reconcile released transfer identity before AutoMigrate: %w", err)
		}
		glog.Info("Released transfer identity reconciled before native startup")
	}
	return nil
}
