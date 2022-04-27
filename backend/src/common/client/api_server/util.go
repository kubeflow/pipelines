package api_server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	apiServerBasePath                  = "/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/"
	apiServerKubeflowInClusterBasePath = "ml-pipeline.%s:8888"
	apiServerDefaultTimeout            = 35 * time.Second
	saDefaultTokenPath                 = "/var/run/secrets/kubeflow/pipelines/token"
	saTokenPathEnvVar                  = "KF_PIPELINES_SA_TOKEN_PATH"
)

// PassThroughAuth never manipulates the request
var PassThroughAuth runtime.ClientAuthInfoWriter = runtime.ClientAuthInfoWriterFunc(
	func(_ runtime.ClientRequest, _ strfmt.Registry) error { return nil })

var SATokenVolumeProjectionAuth runtime.ClientAuthInfoWriter = runtime.ClientAuthInfoWriterFunc(
	func(r runtime.ClientRequest, _ strfmt.Registry) error {
		var projectedPath string
		if projectedPath = os.Getenv(saTokenPathEnvVar); projectedPath == "" {
			projectedPath = saDefaultTokenPath
		}

		content, err := ioutil.ReadFile(projectedPath)
		if err != nil {
			return fmt.Errorf("Failed to read projected SA token at %s: %w", projectedPath, err)
		}

		r.SetHeaderParam("Authorization", "Bearer "+string(content))
		return nil
	})

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

func NewHTTPRuntime(clientConfig clientcmd.ClientConfig, debug bool) (
	*httptransport.Runtime, error) {
	// Creating k8 client
	k8Client, config, namespace, err := util.GetKubernetesClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating K8 client")
	}

	// Create API client
	httpClient := k8Client.RESTClient().(*rest.RESTClient).Client
	masterIPAndPort := util.ExtractMasterIPAndPort(config)
	runtime := httptransport.NewWithClient(masterIPAndPort, fmt.Sprintf(apiServerBasePath, namespace),
		nil, httpClient)

	if debug {
		runtime.SetDebug(true)
	}

	return runtime, err
}

func NewKubeflowInClusterHTTPRuntime(namespace string, debug bool) *httptransport.Runtime {
	schemes := []string{"http"}
	httpClient := http.Client{}
	runtime := httptransport.NewWithClient(fmt.Sprintf(apiServerKubeflowInClusterBasePath, namespace), "/", schemes, &httpClient)
	runtime.SetDebug(debug)
	return runtime
}

func CreateErrorFromAPIStatus(error string, code int32) error {
	return fmt.Errorf("%v (code: %v)", error, code)
}

func CreateErrorCouldNotRecoverAPIStatus(err error) error {
	return fmt.Errorf("Issue calling the service. Use the '--debug' flag to see the HTTP request/response. Raw error from the client: %v",
		err.Error())
}
