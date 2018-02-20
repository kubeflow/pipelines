package dao

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

const (
	argoURLTemplate = "https://%v:%v/apis/argoproj.io/v1alpha1/namespaces/default/%v"
)

type ArgoClientInterface interface {
	Request(method string, api string) ([]byte, error)
}

type ArgoClient struct {
	K8ServiceHost string
	K8TCPPort     string
	K8Token       string
}

func initClient() http.Client {
	// TODO(yangpa): Enable SSL/TLS protection.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return http.Client{Transport: tr}
}

func (ac *ArgoClient) Request(method string, api string) ([]byte, error) {
	client := initClient()

	requestUrl := fmt.Sprintf(argoURLTemplate, ac.K8ServiceHost, ac.K8TCPPort, api)
	req, err := http.NewRequest(method, requestUrl, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", ac.K8Token))

	resp, err := client.Do(req)
	if err != nil {
		glog.Fatalf(err.Error())
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Fatalf(err.Error())
		return nil, err
	}
	return bodyBytes, nil
}
