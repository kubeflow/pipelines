module github.com/kubeflow/pipelines

require (
	github.com/Masterminds/squirrel v0.0.0-20190107164353-fa735ea14f09
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/argoproj/argo v0.0.0-20210125193418-4cb5b7eb8075
	github.com/cenkalti/backoff v2.0.0+incompatible
	github.com/denisenkom/go-mssqldb v0.0.0-20181014144952-4e0d7dc8888f // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.51.1 // indirect
	github.com/go-openapi/errors v0.19.4
	github.com/go-openapi/runtime v0.19.12
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/swag v0.19.8
	github.com/go-openapi/validate v0.19.7
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.2
	github.com/google/addlicense v0.0.0-20200906110928-a0294312aa76 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.1.2
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/jinzhu/gorm v1.9.1
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a // indirect
	github.com/jinzhu/now v0.0.0-20181116074157-8ec929ed50c3 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/moul/http2curl v1.0.0 // indirect
	github.com/peterhellberg/duration v0.0.0-20191119133758-ec6baeebcd10
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	google.golang.org/api v0.20.0
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.29.1
	gopkg.in/gavv/httpexpect.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	k8s.io/code-generator v0.17.9
	k8s.io/kubernetes v1.16.0-alpha.0.0.20190623232353-8c3b7d7679cc
	sigs.k8s.io/controller-runtime v0.5.11
	sigs.k8s.io/testing_frameworks v0.1.1 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.17.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.9
	k8s.io/apiserver => k8s.io/apiserver v0.17.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.9
	k8s.io/client-go => k8s.io/client-go v0.17.9
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.9
	k8s.io/code-generator => k8s.io/code-generator v0.17.9
	k8s.io/component-base => k8s.io/component-base v0.17.9
	k8s.io/cri-api => k8s.io/cri-api v0.17.9
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.9
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.9
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.9
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.9
	k8s.io/kubernetes => k8s.io/kubernetes v1.11.1
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9
)

go 1.13
