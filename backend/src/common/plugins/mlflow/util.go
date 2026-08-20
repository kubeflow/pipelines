// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// maxChildSearchPages bounds SearchRuns pagination in CloseOpenChildRuns.
const maxChildSearchPages = 10

type searchRunPayload struct {
	Info struct {
		RunID   string `json:"run_id"`
		RunUUID string `json:"run_uuid"`
		Status  string `json:"status"`
	} `json:"info"`
}

// IsOpenRunStatus reports whether a run status is non-terminal.
func IsOpenRunStatus(status string) bool {
	switch strings.ToUpper(status) {
	case "FINISHED", "FAILED", "KILLED":
		return false
	default:
		return true
	}
}

// CloseOpenChildRuns closes open direct children of parentRunID. beforeClose
// runs for every valid child (including terminal ones) to reach orphaned
// descendants; UpdateRun is called only for still-open children.
func CloseOpenChildRuns(ctx context.Context, client *Client, experimentID, parentRunID, targetStatus string, endTimeMs *int64, beforeClose func(childRunID string)) []string {
	if client == nil {
		return []string{"MLflow client is required"}
	}
	if experimentID == "" || parentRunID == "" {
		return []string{"experimentID and parentRunID are required to close child MLflow runs"}
	}

	var errs []string
	filter := fmt.Sprintf(`tags.%q = '%s'`, ParentRunTagKey, parentRunID)
	pageToken := ""
	for page := 0; page < maxChildSearchPages; page++ {
		searchResp, err := client.SearchRuns(ctx, []string{experimentID}, filter, 1000, pageToken)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to search child runs of %s: %v", parentRunID, err))
			break
		}
		for _, runPayload := range searchResp.Runs {
			var run searchRunPayload
			if err := json.Unmarshal(runPayload, &run); err != nil {
				errs = append(errs, fmt.Sprintf("failed to decode child run payload: %v", err))
				continue
			}
			childRunID := run.Info.RunID
			if childRunID == "" {
				childRunID = run.Info.RunUUID
			}
			if childRunID == "" || childRunID == parentRunID {
				continue
			}
			if beforeClose != nil {
				beforeClose(childRunID)
			}
			if !IsOpenRunStatus(run.Info.Status) {
				continue
			}
			if err := client.UpdateRun(ctx, childRunID, targetStatus, endTimeMs); err != nil {
				errs = append(errs, fmt.Sprintf("failed to close child run %s: %v", childRunID, err))
			}
		}
		if searchResp.NextPageToken == "" {
			break
		}
		if page == maxChildSearchPages-1 {
			errs = append(errs, fmt.Sprintf(
				"pagination limit (%d pages) reached while closing child runs of %s; some runs may have been skipped",
				maxChildSearchPages, parentRunID))
			break
		}
		pageToken = searchResp.NextPageToken
	}
	return errs
}
