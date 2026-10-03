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

// run-history transfers completed native KFP history using administrator DB access.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/history"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const maxArchiveBytes = 128 << 20

type runIDs []string

func (r *runIDs) String() string { return strings.Join(*r, ",") }
func (r *runIDs) Set(value string) error {
	if value == "" {
		return errors.New("run ID must not be empty")
	}
	*r = append(*r, value)
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:], os.Stdout)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "run-history:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 || (args[0] != "export" && args[0] != "import") {
		return errors.New("usage: run-history export|import --file ARCHIVE [options]; use --help after the command")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	driver := flags.String("driver", "mysql", "database driver: mysql or postgres (use the branch's supported backend)")
	dsnEnv := flags.String("dsn-env", "KFP_HISTORY_DSN", "environment variable containing the database DSN; never pass credentials on the command line")
	file := flags.String("file", "", "JSON archive path (export refuses to overwrite)")
	var source, prefix, experiment string
	var ids runIDs
	var apply bool
	if args[0] == "export" {
		flags.StringVar(&source, "source-id", "", "stable source installation ID")
		flags.Var(&ids, "run-id", "completed run ID; repeat up to 100 times")
	} else {
		flags.BoolVar(&apply, "apply", false, "commit the import; without this flag validate in a rolled-back transaction")
		flags.StringVar(&prefix, "name-prefix", "", "prefix imported experiment/pipeline names to avoid collisions")
		flags.StringVar(&experiment, "experiment-id", "", "explicit existing target experiment in the same namespace")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *file == "" {
		return errors.New("--file is required and positional arguments are not accepted")
	}
	dsn := os.Getenv(*dsnEnv)
	if dsn == "" {
		return fmt.Errorf("database DSN environment variable %s is empty", *dsnEnv)
	}
	var dialector gorm.Dialector
	switch *driver {
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return errors.New("driver must be mysql or postgres")
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if args[0] == "export" {
		bundle, err := history.Export(ctx, db, source, ids)
		if err != nil {
			return err
		}
		if err := writeArchive(*file, bundle); err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Exported %d completed runs. Artifact files and logs are not included.\n", len(bundle.Entries))
		return err
	}
	bundle, err := readArchive(*file)
	if err != nil {
		return err
	}
	result, err := history.Import(ctx, db, bundle, history.ImportOptions{NamePrefix: prefix, ExperimentID: experiment, DryRun: !apply})
	if err != nil {
		return err
	}
	mode := "Validated (rolled back)"
	if apply {
		mode = "Imported"
	}
	_, err = fmt.Fprintf(out, "%s: %d new runs, %d already imported.\n", mode, result.Imported, result.Skipped)
	return err
}

func writeArchive(path string, bundle *history.Bundle) error {
	data, err := json.Marshal(bundle)
	if err != nil {
		return err
	}
	if len(data) > maxArchiveBytes {
		return errors.New("archive exceeds 128 MiB; export fewer runs")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		file.Close()
		if !complete {
			os.Remove(path)
		}
	}()
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

func readArchive(path string) (*history.Bundle, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxArchiveBytes {
		return nil, errors.New("archive exceeds 128 MiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var bundle history.Bundle
	if err := decoder.Decode(&bundle); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("archive must contain exactly one JSON object")
	}
	return &bundle, nil
}
