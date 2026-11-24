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
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type headers []string

func (values *headers) String() string         { return strings.Join(*values, ",") }
func (values *headers) Set(value string) error { *values = append(*values, value); return nil }

func run() error {
	var address, dsn, namespace, deployment, recurringDeployment string
	var useTLS, drainCheck, gateOnly, capacityOnly, sourceSingleUser bool
	var budget mlmd.DiskBudget
	var extraHeaders headers
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "--check-release-eligibility" {
		if !mlmd.CutoverEnabled {
			return errors.New(mlmd.CutoverHoldMessage)
		}
		return nil
	}
	validate := len(args) != 0 && args[0] == "validate"
	if validate {
		args = args[1:]
	}
	flags := flag.NewFlagSet("mlmd-migrate [validate]", flag.ContinueOnError)
	flags.StringVar(&address, "mlmd-address", os.Getenv("KFP_MLMD_ADDRESS"), "MLMD gRPC host:port")
	flags.StringVar(&dsn, "mysql-dsn", "", "MySQL driver DSN for the KFP database")
	flags.StringVar(&namespace, "namespace", "kubeflow", "namespace of the API Deployment")
	flags.StringVar(&deployment, "deployment", "ml-pipeline", "API Deployment to fence")
	flags.StringVar(&recurringDeployment, "recurring-deployment", "ml-pipeline-scheduledworkflow", "recurring-run controller Deployment to pause")
	flags.BoolVar(&sourceSingleUser, "source-single-user", false, "allow verified stored Workflow ownership for a legacy single-user source")
	flags.BoolVar(&useTLS, "mlmd-tls", false, "use TLS for the MLMD connection")
	flags.BoolVar(&drainCheck, "drain-check", false, "verify terminal KFP runs and Workflow reports before fencing old replicas")
	flags.BoolVar(&gateOnly, "gate-only", false, "verify the full writer fence without migrating")
	flags.Var(&extraHeaders, "mlmd-header", "gRPC header as key:value (repeatable)")
	flags.Uint64Var(&budget.AvailableBytes, "available-disk-bytes", 0, "free bytes on the destination MySQL data volume")
	flags.Uint64Var(&budget.MLMDDatabaseBytes, "mlmd-database-bytes", 0, "allocated bytes of the source MLMD database")
	flags.BoolVar(&capacityOnly, "check-capacity", false, "report the read-only destination disk capacity check")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return nil
	} else if err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if validate && (drainCheck || gateOnly || capacityOnly) {
		return fmt.Errorf("validate cannot be combined with a check mode")
	}
	if !mlmd.CutoverEnabled && !validate && !drainCheck && !gateOnly && !capacityOnly {
		return errors.New(mlmd.CutoverHoldMessage)
	}
	if dsn == "" {
		dsn = os.Getenv("KFP_MLMD_MYSQL_DSN")
	}
	if dsn == "" {
		return fmt.Errorf("--mysql-dsn or KFP_MLMD_MYSQL_DSN is required")
	}
	if (drainCheck && gateOnly) || (capacityOnly && (drainCheck || gateOnly)) {
		return fmt.Errorf("choose at most one check mode")
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return fmt.Errorf("legacy MLMD migration requires MySQL; PostgreSQL upgrades are unsupported")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open KFP MySQL database: %w", err)
	}
	defer database.Close()
	if capacityOnly {
		report, err := mlmd.CheckDiskCapacity(ctx, database, budget)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(report)
	}

	var gate mlmd.DrainGate
	var state mlmd.StartupState
	if !drainCheck && !gateOnly && !validate {
		state, err = mlmd.ClassifyStartup(ctx, database, "mysql")
		if err != nil {
			return err
		}
		if state == mlmd.MigrationRequired && address == "" {
			return fmt.Errorf("incomplete MLMD migration requires --mlmd-address")
		}
	}
	checkFence := state == mlmd.MigrationRequired || state == mlmd.CleanupPending
	if checkFence || drainCheck || gateOnly || validate {
		kubeConfig, err := util.GetKubernetesConfig()
		if err != nil {
			return fmt.Errorf("load Kubernetes config for writer fence: %w", err)
		}
		kubeConfig.Timeout = 30 * time.Second
		workflowClient, err := dynamic.NewForConfig(kubeConfig)
		if err != nil {
			return err
		}
		gate = mlmd.DrainGate{Database: database, Workflows: workflowClient, Namespace: namespace, Deployment: deployment, RecurringDeployment: recurringDeployment, SourceSingleUser: sourceSingleUser}
		if checkFence || gateOnly || sourceSingleUser {
			gate.Kubernetes, err = kubernetes.NewForConfig(kubeConfig)
			if err != nil {
				return err
			}
		}
		if drainCheck {
			if err := gate.CheckDrained(ctx); err != nil {
				return fmt.Errorf("drain check failed: %w", err)
			}
			return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "drained"})
		}
		if checkFence || gateOnly {
			if err := gate.Check(ctx); err != nil {
				return fmt.Errorf("migration refused: %w", err)
			}
			if gateOnly {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "fenced"})
			}
		}
	}

	var source pb.MetadataStoreServiceClient
	if address != "" && (validate || state == mlmd.MigrationRequired) {
		connection, err := mlmd.DialSource(address, useTLS)
		if err != nil {
			return err
		}
		defer connection.Close()
		metadataValues := metadata.MD{}
		for _, value := range extraHeaders {
			key, content, found := strings.Cut(value, ":")
			if !found || key == "" {
				return fmt.Errorf("invalid --mlmd-header: expected key:value")
			}
			metadataValues.Append(key, strings.TrimSpace(content))
		}
		ctx = metadata.NewOutgoingContext(ctx, metadataValues)
		source = pb.NewMetadataStoreServiceClient(connection)
	}
	if validate {
		validationCtx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		sourceErr := mlmd.ValidateSource(validationCtx, database, source)
		blockers, drainErr := gate.Blockers(validationCtx)
		status := "validated"
		if sourceErr != nil || drainErr != nil || len(blockers) != 0 {
			status = "blocked"
		}
		outputErr := json.NewEncoder(os.Stdout).Encode(struct {
			Status   string   `json:"status"`
			Blockers []string `json:"blockers,omitempty"`
		}{status, blockers})
		if len(blockers) != 0 {
			drainErr = errors.Join(drainErr, fmt.Errorf("%d active runs or Workflow blockers", len(blockers)))
		}
		return errors.Join(sourceErr, drainErr, outputErr)
	}
	report, err := mlmd.MigrateFenced(ctx, source, database, budget, gate)
	if err != nil {
		if len(report.Skipped) > 0 || len(report.Warnings) > 0 {
			if encodeErr := json.NewEncoder(os.Stderr).Encode(report); encodeErr != nil {
				return fmt.Errorf("migration incomplete: %w (also failed to write diagnostics: %v)", err, encodeErr)
			}
		}
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
