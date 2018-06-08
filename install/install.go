package install

import (
	"path/filepath"
	"os"
	"io"
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	log "github.com/sirupsen/logrus"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type InstallOption struct {
	Namespace       string // --namespace
	ControllerImage string // --controller-image
	ImagePullPolicy string // --image-pull-policy
}

type Installer struct {
	InstallOption
	config *rest.Config
	clientset *kubernetes.Clientset
	apiextcsClientset apiextcs.Interface
}

func NewInstaller(config *rest.Config, installOption InstallOption) (*Installer, error) {
	installer := &Installer{
		InstallOption: installOption,
		config: config,
	}
	var err error
	installer.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	installer.apiextcsClientset, err = apiextcs.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return installer, nil
}

func (i *Installer) Install() {
	i.installScheduledWorkflowCRD()
	i.installScheduledWorkflowController()
}

func (i *Installer) installScheduledWorkflowCRD() {
	absPath, err := filepath.Abs("install/manifests/scheduledworkflow-crd.yaml")
	if err != nil {
		log.Fatal(err)
	}

	manifest, err := os.Open(absPath)
	if err != nil {
		log.Fatal(err)
	}

	decoder := yaml.NewYAMLOrJSONDecoder(manifest, 4096)
	for {
		var obj unstructured.Unstructured
		err = decoder.Decode(&obj)
		if err != nil {
			break
		}

		switch obj.GetKind() {
		case "CustomResourceDefinition":
			var crd apiextensionsv1beta1.CustomResourceDefinition
			marshaled, _ := obj.MarshalJSON()
			json.Unmarshal(marshaled, &crd)
			i.apiextcsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(&crd)
		case "Deployment":
			var deployment v1beta1.Deployment
			marshaled, _ := obj.MarshalJSON()
			json.Unmarshal(marshaled, deployment)
			i.clientset.Extensions().Deployments(i.Namespace).Create(&deployment)
		default:
			// 
		}
	}

	if err != io.EOF && err != nil {
		log.Fatal(err)
	}
}

func (i *Installer) installScheduledWorkflowController() {
	
}
