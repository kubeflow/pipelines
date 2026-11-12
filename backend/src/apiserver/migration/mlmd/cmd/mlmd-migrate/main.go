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
	var useTLS bool
	var extraHeaders headers
	flag.StringVar(&address, "mlmd-address", "", "MLMD gRPC host:port")
	flag.StringVar(&dsn, "mysql-dsn", "", "MySQL driver DSN for the KFP database")
	flag.BoolVar(&useTLS, "mlmd-tls", false, "use TLS for the MLMD connection")
	flag.Var(&extraHeaders, "mlmd-header", "gRPC header as key:value (repeatable)")
	flag.Parse()
	if address == "" || dsn == "" {
		return fmt.Errorf("--mlmd-address and --mysql-dsn are required")
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
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
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open KFP MySQL database: %w", err)
	}
	defer database.Close()
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
