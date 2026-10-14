// Scheduled workload stage. Run outside the shared measured load clock.
package main

import (
	"context"
	"encoding/json"
	"flag"
	transport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	rc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client"
	rp "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	rm "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	uc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client"
	up "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"net/url"
	"os"
	"strings"
	"time"
)

type Fixture struct {
	Name       string `json:"name"`
	PipelineID string `json:"pipeline_id"`
	VersionID  string `json:"version_id"`
}
type Tenant struct {
	Name           string    `json:"name"`
	Namespace      string    `json:"namespace"`
	ExperimentID   string    `json:"experiment_id"`
	ServiceAccount string    `json:"service_account"`
	TokenFile      string    `json:"token_file"`
	Fixtures       []Fixture `json:"fixtures"`
}
type Config struct {
	Endpoint string   `json:"endpoint"`
	Label    string   `json:"label"`
	Tenants  []Tenant `json:"tenants"`
}
type Result struct {
	Tenant         string `json:"tenant"`
	RecurringID    string `json:"recurring_run_id,omitempty"`
	ChildID        string `json:"child_run_id,omitempty"`
	Created        bool   `json:"created"`
	Disabled       bool   `json:"disabled"`
	ChildSucceeded bool   `json:"child_succeeded"`
	Complete       bool   `json:"complete"`
	Failure        string `json:"failure,omitempty"`
}

func call(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 10*time.Second)
}
func stage(parent context.Context, endpoint, label string, t Tenant, token string, budget time.Duration, poll time.Duration, save func(Result) error) (result Result) {
	result.Tenant = t.Name
	record := func() bool {
		if save(result) != nil {
			result.Failure = "capture_failure"
			return false
		}
		return true
	}
	u, e := url.Parse(endpoint)
	if e != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		result.Failure = "invalid_endpoint"
		record()
		return
	}
	if t.Name == "" || t.Namespace == "" || t.ExperimentID == "" || t.ServiceAccount == "" || len(t.Fixtures) == 0 || t.Fixtures[0].PipelineID == "" || t.Fixtures[0].VersionID == "" || token == "" {
		result.Failure = "invalid_tenant"
		record()
		return
	}
	tr := transport.New(u.Host, u.Path, []string{u.Scheme})
	tr.DefaultAuthentication = transport.BearerToken(token)
	recurring := rc.New(tr, strfmt.Default)
	runs := uc.New(tr, strfmt.Default)
	ctx, cancel := context.WithTimeout(parent, budget)
	defer cancel()
	f := t.Fixtures[0]
	now := time.Now().UTC()
	// Interval exceeds the entire stage: at most one scheduled occurrence; end bounds ambiguous create too.
	body := &rm.V2beta1RecurringRun{DisplayName: "apf-scheduled-" + label + "-" + t.Name, Namespace: t.Namespace, ExperimentID: t.ExperimentID, ServiceAccount: t.ServiceAccount, MaxConcurrency: 1, NoCatchup: true, Mode: rm.RecurringRunModeENABLE.Pointer(), PipelineVersionReference: &rm.V2beta1PipelineVersionReference{PipelineID: f.PipelineID, PipelineVersionID: f.VersionID}, Trigger: &rm.V2beta1Trigger{PeriodicSchedule: &rm.V2beta1PeriodicSchedule{StartTime: strfmt.DateTime(now), EndTime: strfmt.DateTime(now.Add(budget)), IntervalSecond: int64(budget.Seconds()) + 60}}}
	if !record() {
		return
	}
	c, done := call(ctx)
	created, err := recurring.RecurringRunService.RecurringRunServiceCreateRecurringRunContext(c, &rp.RecurringRunServiceCreateRecurringRunParams{RecurringRun: body})
	done()
	if err != nil || created == nil || created.Payload == nil || created.Payload.RecurringRunID == "" {
		result.Failure = "create_failed_or_ambiguous_no_retry"
		record()
		return
	}
	result.Created = true
	result.RecurringID = created.Payload.RecurringRunID
	// One disable attempt even if capture/read fails. Fresh bounded context for cleanup; owner token only.
	defer func() {
		c, done := call(context.Background())
		defer done()
		_, e := recurring.RecurringRunService.RecurringRunServiceDisableRecurringRunContext(c, &rp.RecurringRunServiceDisableRecurringRunParams{RecurringRunID: result.RecurringID})
		if e != nil {
			result.Failure = "disable_failed_no_retry"
		} else {
			result.Disabled = true
			c2, d2 := call(context.Background())
			defer d2()
			got, e2 := recurring.RecurringRunService.RecurringRunServiceGetRecurringRunContext(c2, &rp.RecurringRunServiceGetRecurringRunParams{RecurringRunID: result.RecurringID})
			if e2 != nil || got == nil || got.Payload == nil || got.Payload.Status == nil || *got.Payload.Status != rm.V2beta1RecurringRunStatusDISABLED {
				result.Disabled = false
				result.Failure = "disable_not_confirmed"
			}
		}
		result.Complete = result.ChildSucceeded && result.Disabled && result.Failure == ""
		record()
	}()
	if !record() {
		return
	}
	for ctx.Err() == nil {
		size := int32(100)
		page := ""
		seen := map[string]bool{}
		found := false
		for n := 0; n < 10; n++ {
			c, done := call(ctx)
			resp, e := runs.RunService.RunServiceListRunsContext(c, &up.RunServiceListRunsParams{Namespace: &t.Namespace, ExperimentID: &t.ExperimentID, PageSize: &size, PageToken: &page}, transport.BearerToken(token))
			done()
			if e != nil || resp == nil || resp.Payload == nil {
				result.Failure = "list_failed"
				record()
				return
			}
			for _, r := range resp.Payload.Runs {
				if r.RecurringRunID == result.RecurringID {
					if r.Namespace != t.Namespace || r.ExperimentID != t.ExperimentID || r.RunID == "" {
						result.Failure = "child_ownership_mismatch"
						record()
						return
					}
					result.ChildID = r.RunID
					found = true
					if r.State != nil {
						state := string(*r.State)
						if state == "SUCCEEDED" {
							result.ChildSucceeded = true
							record()
							return
						}
						if state == "FAILED" || state == "CANCELED" {
							result.Failure = "child_terminal_failure"
							record()
							return
						}
					}
				}
			}
			page = resp.Payload.NextPageToken
			if page == "" {
				break
			}
			if seen[page] || n == 9 {
				result.Failure = "pagination_bound"
				record()
				return
			}
			seen[page] = true
		}
		if found && !record() {
			return
		}
		select {
		case <-ctx.Done():
		case <-time.After(poll):
		}
	}
	result.Failure = "stage_deadline"
	record()
	return
}
func main() {
	config := flag.String("config", "", "same tenant config as workload")
	out := flag.String("out", "", "new result JSONL file")
	seconds := flag.Int("seconds", 600, "per-tenant stage bound 30..600")
	flag.Parse()
	if *seconds < 30 || *seconds > 600 {
		os.Exit(2)
	}
	b, e := os.ReadFile(*config)
	var cfg Config
	if e != nil || json.Unmarshal(b, &cfg) != nil || len(cfg.Tenants) < 2 || len(cfg.Tenants) > 4 {
		os.Exit(2)
	}
	f, e := os.OpenFile(*out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		os.Exit(2)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	save := func(r Result) error {
		if e := enc.Encode(r); e != nil {
			return e
		}
		return f.Sync()
	}
	ok := true
	for _, t := range cfg.Tenants {
		tok, e := os.ReadFile(t.TokenFile)
		if e != nil {
			ok = false
			break
		}
		r := stage(context.Background(), cfg.Endpoint, cfg.Label, t, strings.TrimSpace(string(tok)), time.Duration(*seconds)*time.Second, time.Second*2, save)
		if !r.Complete {
			ok = false
			break
		}
	}
	if !ok {
		os.Exit(1)
	}
}
