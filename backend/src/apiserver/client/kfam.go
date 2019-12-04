package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

type KFAMInterface interface {
	IsAuthorized(userIdentity string, namespace string) (bool, error)
}

type KFAMClient struct {
	kfamServiceUrl string
}

type User struct {
	Kind string
	Name string
}

type RoleRef struct {
	ApiGroup string
	Kind     string
	Name     string
}

type Binding struct {
	User              User
	ReferredNamespace string
	RoleRef           RoleRef
}

type Bindings struct {
	Bindings []Binding
}

func (c *KFAMClient) IsAuthorized(userIdentity string, namespace string) (bool, error) {
	req, err := http.NewRequest("GET", c.kfamServiceUrl, nil)
	if err != nil {
		return false, util.NewInternalServerError(err, "Failed to create a KFAM http request.")
	}
	q := req.URL.Query()
	q.Add("user", userIdentity)
	req.URL.RawQuery = q.Encode()
	resp, err := http.Get(req.URL.String())
	if err != nil {
		return false, util.NewInternalServerError(err, "Failed to connect to the KFAM service.")
	}
	if resp.StatusCode != http.StatusOK {
		return false, util.NewInternalServerError(errors.New("Requests to the KFAM service failed."), resp.Status)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, util.NewInternalServerError(err, "Unable to parse KFAM response.")
	}
	glog.Info(string(body))
	var jsonBindings Bindings
	err = json.Unmarshal(body, &jsonBindings)
	if err != nil {
		return false, util.NewInternalServerError(err, "Failed to parse KFAM response.")
	}
	nsFound := false
	for _, jsonBinding := range jsonBindings.Bindings {
		if jsonBinding.ReferredNamespace == namespace {
			nsFound = true
			break
		}
	}
	return nsFound, nil
}

func NewKFAMClient(kfamServiceHost string, kfamServicePort string) *KFAMClient {
	kfamServiceUrl := fmt.Sprintf("http://%s:%s/kfam/v1/bindings", kfamServiceHost, kfamServicePort)
	return &KFAMClient{kfamServiceUrl}
}
