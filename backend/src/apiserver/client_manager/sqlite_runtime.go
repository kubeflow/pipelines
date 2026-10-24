// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientmanager

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// sqliteRuntimeDSN acquires write intent before store transactions read their
// snapshots. A WAL read transaction cannot wait its way out of SQLITE_BUSY_SNAPSHOT
// when another connection commits before it upgrades to a writer.
func sqliteRuntimeDSN(dsn string) (string, error) {
	path, query, _ := strings.Cut(dsn, "?")
	options, err := url.ParseQuery(query)
	if err != nil {
		return "", fmt.Errorf("parse SQLite runtime connection options: %w", err)
	}
	if value := options.Get("_txlock"); value != "" && value != "immediate" {
		return "", fmt.Errorf("SQLite runtime requires _txlock=immediate, not %q; remove the conflicting connection option", value)
	}
	options.Set("_txlock", "immediate")
	foreignKeys := options.Get("_foreign_keys")
	if foreignKeys == "" {
		foreignKeys = options.Get("_fk")
	}
	switch strings.ToLower(foreignKeys) {
	case "", "1", "on", "true", "yes":
		options.Del("_fk")
		options.Set("_foreign_keys", "on")
	default:
		return "", fmt.Errorf("SQLite runtime requires foreign key enforcement; remove _foreign_keys=%q", foreignKeys)
	}
	if path != ":memory:" && path != "file::memory:" && options.Get("mode") != "memory" {
		// Shared-cache SQLITE_LOCKED errors bypass the busy handler. WAL
		// file databases share data, not connection page-cache locks.
		options.Set("cache", "private")
	}
	timeout := options.Get("_busy_timeout")
	if timeout == "" {
		timeout = options.Get("_timeout")
	}
	if timeout == "" {
		timeout = "5000"
	}
	milliseconds, err := strconv.ParseInt(timeout, 10, 64)
	if err != nil || milliseconds <= 0 || milliseconds > 30000 {
		return "", fmt.Errorf("SQLite runtime busy timeout must be between 1 and 30000 milliseconds, found %q", timeout)
	}
	options.Del("_timeout")
	options.Set("_busy_timeout", timeout)
	return path + "?" + options.Encode(), nil
}
