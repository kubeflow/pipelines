package api_server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	testconfig "github.com/kubeflow/pipelines/backend/test/config"
	"github.com/pkg/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	APIServerDefaultTimeout            = 35 * time.Second
	apiServerBasePath                  = "/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/"
	apiServerKubeflowInClusterBasePath = "ml-pipeline.%s:8888"
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

		content, err := os.ReadFile(projectedPath)
		if err != nil {
			return fmt.Errorf("Failed to read projected SA token at %s: %w", projectedPath, err)
		}

		r.SetHeaderParam("Authorization", "Bearer "+string(content))
		return nil
	})

func TokenToAuthInfo(userToken string) runtime.ClientAuthInfoWriter {
	return runtime.ClientAuthInfoWriterFunc(
		func(r runtime.ClientRequest, _ strfmt.Registry) error {
			err := r.SetHeaderParam("Authorization", "Bearer "+userToken)
			if err != nil {
				return err
			}
			return nil
		})
}

func NewHTTPRuntime(clientConfig clientcmd.ClientConfig, debug bool, tlsCfg *tls.Config) (
	*httptransport.Runtime, error,
) {
	if !*testconfig.InClusterRun {
		httpClient := http.DefaultClient
		var scheme []string
		parsedUrl, err := url.Parse(*testconfig.ApiUrl)
		if err != nil {
			return nil, err
		}
		host := parsedUrl.Host
		if parsedUrl.Scheme != "" {
			scheme = []string{parsedUrl.Scheme}
		} else {
			if testconfig.ApiScheme != nil {
				scheme = []string{*testconfig.ApiScheme}
			} else {
				scheme = []string{"http"}
			}
		}
		if *testconfig.DisableTLSCheck {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			httpClient = &http.Client{Transport: tr}
		}
		var runtimeClient *httptransport.Runtime
		if tlsCfg != nil && !*testconfig.DisableTLSCheck {
			scheme = []string{"https"}
			tr := &http.Transport{
				TLSClientConfig: tlsCfg,
			}
			httpClient = &http.Client{Transport: tr}
		}
		runtimeClient = httptransport.NewWithClient(host, "", scheme, httpClient)
		if debug {
			runtimeClient.SetDebug(true)
		}
		return runtimeClient, nil
	}

	// Creating k8 client
	k8Client, config, namespace, err := util.GetKubernetesClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating K8 client")
	}

	// Create API client
	httpClient := k8Client.RESTClient().(*rest.RESTClient).Client
	masterIPAndPort := util.ExtractMasterIPAndPort(config)
	runtimeClient := httptransport.NewWithClient(masterIPAndPort, fmt.Sprintf(apiServerBasePath, namespace),
		nil, httpClient)

	if debug {
		runtimeClient.SetDebug(true)
	}

	return runtimeClient, err
}

func NewKubeflowInClusterHTTPRuntime(namespace string, debug bool, tlsCfg *tls.Config) *httptransport.Runtime {
	var schemes []string
	var httpClient *http.Client
	if tlsCfg != nil {
		schemes = []string{"https"}
		tr := &http.Transport{TLSClientConfig: tlsCfg}
		httpClient = &http.Client{Transport: tr}
	} else {
		schemes = []string{"http"}
		httpClient = &http.Client{}
	}
	runtimeClient := httptransport.NewWithClient(
		fmt.Sprintf(apiServerKubeflowInClusterBasePath, namespace), "/", schemes, httpClient)
	runtimeClient.SetDebug(debug)
	return runtimeClient
}

func CreateErrorFromAPIStatus(error string, code int32) error {
	return fmt.Errorf("%v (code: %v)", error, code)
}

func CreateErrorCouldNotRecoverAPIStatus(err error) error {
	return fmt.Errorf("issue calling the service. Use the '--debug' flag to see the HTTP request/response. Raw error from the client: %v",
		err.Error())
}
