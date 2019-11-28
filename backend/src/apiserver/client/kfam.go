package client

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type KFAMInterface interface {
	IsAuthorized(userIdentity string, namespace string) (bool, error)
}

type KFAMClient struct {
	kfamServiceUrl     string
}


func (c *KFAMClient) IsAuthorized(userIdentity string, namespace string) (bool, error) {
	req, err := http.NewRequest("GET", c.kfamServiceUrl, nil)
	if err != nil {
		return false, util.Wrap(err, "Failed to create a KFAM http request.")
	}
	q := req.URL.Query()
	q.Add("user", userIdentity)
	req.URL.RawQuery = q.Encode()
	resp, err := http.Get(req.URL.String())
	if err != nil {
		return false, util.Wrap(err, "Failure to connect to the KFAM service.")
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf(resp.Status)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, util.Wrap(err, "Unable to parse KFAM response.")
	}
	glog.Info(string(body))
	return true, nil
}

func NewKFAMClient(kfamServiceHost string, kfamServicePort string) *KFAMClient{
	kfamServiceUrl := fmt.Sprintf("http://%s:%s/kfam/v1/bindings", kfamServiceHost, kfamServicePort)
	return &KFAMClient{kfamServiceUrl}
}