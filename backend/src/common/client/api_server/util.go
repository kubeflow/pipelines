package api_server

import (
	"time"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	clientmodel "github.com/googleprivate/ml/backend/api/go_http_client/run_model"
	servermodel "github.com/googleprivate/ml/backend/src/apiserver/model"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	apiServerBasePath       = "/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/"
	apiServerDefaultTimeout = 35 * time.Second
)

// PassThroughAuth never manipulates the request
var PassThroughAuth runtime.ClientAuthInfoWriter = runtime.ClientAuthInfoWriterFunc(
	func(_ runtime.ClientRequest, _ strfmt.Registry) error { return nil })

func toClientJobDetail(runDetail *servermodel.RunDetail) *clientmodel.APIRunDetail {
	return &clientmodel.APIRunDetail{
		Run:      toClientAPIRun(runDetail.Run),
		Workflow: runDetail.WorkflowRuntimeManifest,
	}
}

func toClientAPIRun(run servermodel.Run) *clientmodel.APIRun {
	return &clientmodel.APIRun{
		CreatedAt:   toDateTimeTestOnly(run.CreatedAtInSec),
		ID:          run.JobID,
		Name:        run.Name,
		ScheduledAt: toDateTimeTestOnly(run.ScheduledAtInSec),
		Status:      run.Conditions,
	}
}

func toDateTimeTestOnly(timeInSec int64) strfmt.DateTime {
	result, err := strfmt.ParseDateTime(time.Unix(timeInSec, 0).String())
	if err != nil {
		return strfmt.NewDateTime()
	}
	return result
}

func toWorkflowTestOnly(workflow string) *workflowapi.Workflow {
	var result workflowapi.Workflow
	err := yaml.Unmarshal([]byte(workflow), &result)
	if err != nil {
		return nil
	}
	return &result
}
