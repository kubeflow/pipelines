package main

import (
	"context"
	"database/sql"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRuntimeCleanupCommandDoesNotStartServicesOrAdmitTasks(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "artifacts"), 0o700))
	// No Node, shell utilities, or real Docker daemon are available on PATH.
	require.NoError(t, os.WriteFile(filepath.Join(root, "docker"), []byte(
		"#!/bin/sh\ncase \"$1\" in info) printf '28\\n';; *) exit 99;; esac\n",
	), 0o700))
	t.Setenv("PATH", root)
	for key, value := range map[string]any{
		common.V2RuntimeMode:                    "coordinator-poc",
		common.V2RuntimeExecutor:                "docker",
		common.V2RuntimeAutoExecute:             true,
		common.V2RuntimeWorkspaceRoot:           filepath.Join(root, "runtime"),
		"DBDriverName":                          "sqlite",
		"DBConfig.SQLiteConfig.DataSourceName":  "file:" + filepath.Join(root, "metadata.sqlite"),
		"DBConfig.ConMaxLifeTime":               "120s",
		"InitConnectionTimeout":                 "10s",
		"ObjectStoreConfig.BucketURL":           "file://" + filepath.Join(root, "artifacts"),
		"ObjectStoreConfig.PipelinePath":        "pipelines",
		"MultiUser":                             false,
		"SCHEDULED_WORKFLOW_CONTROLLER_ENABLED": false,
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	previousHTTP, previousRPC := *httpPortFlag, *rpcPortFlag
	previousTimeout := *runtimeCleanupTimeout
	*httpPortFlag, *rpcPortFlag = listener.Addr().String(), listener.Addr().String()
	*runtimeCleanupTimeout = 10 * time.Second
	t.Cleanup(func() {
		*httpPortFlag, *rpcPortFlag = previousHTTP, previousRPC
		*runtimeCleanupTimeout = previousTimeout
	})
	require.NoError(t, cleanupRuntimeCommand(context.Background()))
	db, err := sql.Open("sqlite3", filepath.Join(root, "metadata.sqlite"))
	require.NoError(t, err)
	defer db.Close()
	for _, table := range []string{"run_details", "tasks", "runtime_tasks", "experiments"} {
		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count, "cleanup must not create runs, tasks, or a default experiment")
	}
	require.NoError(t, cleanupRuntimeCommand(context.Background()), "the command must release its worker lock")
}
