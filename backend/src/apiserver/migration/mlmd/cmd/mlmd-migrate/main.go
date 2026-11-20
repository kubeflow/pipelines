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
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type headers []string

func (values *headers) String() string         { return strings.Join(*values, ",") }
func (values *headers) Set(value string) error { *values = append(*values, value); return nil }

func run() error {
	var address, dsn string
	var useTLS, capacityOnly bool
	var budget mlmd.DiskBudget
	var extraHeaders headers
	flag.StringVar(&address, "mlmd-address", "", "MLMD gRPC host:port")
	flag.StringVar(&dsn, "mysql-dsn", "", "MySQL driver DSN for the KFP database")
	flag.BoolVar(&useTLS, "mlmd-tls", false, "use TLS for the MLMD connection")
	flag.Var(&extraHeaders, "mlmd-header", "gRPC header as key:value (repeatable)")
	flag.Uint64Var(&budget.AvailableBytes, "available-disk-bytes", 0, "free bytes on the destination MySQL data volume")
	flag.Uint64Var(&budget.MLMDDatabaseBytes, "mlmd-database-bytes", 0, "allocated bytes of the source MLMD database")
	flag.BoolVar(&capacityOnly, "check-capacity", false, "report the read-only destination disk capacity check")
	flag.Parse()
	if dsn == "" {
		return fmt.Errorf("--mysql-dsn is required")
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

	var source pb.MetadataStoreServiceClient
	if address != "" {
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
				return fmt.Errorf("invalid --mlmd-header: expected key:value")
			}
			metadataValues.Append(key, strings.TrimSpace(content))
		}
		ctx = metadata.NewOutgoingContext(ctx, metadataValues)
		source = pb.NewMetadataStoreServiceClient(connection)
	}
	report, err := mlmd.Migrate(ctx, source, database, budget)
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
