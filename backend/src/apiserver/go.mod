module apiserver

go 1.15

require (
	bou.ke/staticfiles v0.0.0-20190225145250-827d7f6389cd // indirect
	cloud.google.com/go v0.55.0 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/Masterminds/squirrel v1.4.0
	github.com/VividCortex/mysqlerr v0.0.0-20200629151747-c28746d985dd
	github.com/aliyun/aliyun-oss-go-sdk v2.0.6+incompatible // indirect
	github.com/antonmedv/expr v1.8.2 // indirect
	github.com/argoproj/argo v2.5.2+incompatible
	github.com/argoproj/pkg v0.1.0 // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/colinmarc/hdfs v1.1.4-0.20180805212432-9746310a4d31 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gavv/httpexpect/v2 v2.0.3 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/zapr v0.1.0 // indirect
	github.com/go-openapi/spec v0.19.8 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-swagger/go-swagger v0.23.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.2
	github.com/google/addlicense v0.0.0-20200422172452-68a83edd47bc // indirect
	github.com/gophercloud/gophercloud v0.7.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.14.8
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jinzhu/gorm v1.9.16
	github.com/klauspost/compress v1.9.7 // indirect
	github.com/kubeflow/pipelines v0.0.0-00010101000000-000000000000
	github.com/lib/pq v1.3.0 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/mattn/go-sqlite3 v1.14.2
	github.com/mattn/goreman v0.3.5 // indirect
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b // indirect
	github.com/peterhellberg/duration v0.0.0-20191119133758-ec6baeebcd10 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/robfig/cron v1.2.0
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.3.5 // indirect
	github.com/valyala/fasttemplate v1.1.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/tools v0.0.0-20200630154851-b2d8b0336632 // indirect
	gonum.org/v1/netlib v0.0.0-20190331212654-76723241ea4e // indirect
	google.golang.org/grpc v1.32.0
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/go-playground/webhooks.v5 v5.15.0 // indirect
	gopkg.in/jcmturner/goidentity.v2 v2.0.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kubernetes v1.11.1 // indirect
	k8s.io/utils v0.0.0-20200821003339-5e75c0163111 // indirect
	sigs.k8s.io/controller-runtime v0.0.0-20181121180216-5558165425ef // indirect
	sigs.k8s.io/controller-tools v0.3.0 // indirect
	sigs.k8s.io/testing_frameworks v0.1.1 // indirect
	upper.io/db.v3 v3.6.3+incompatible // indirect
)

replace github.com/argoproj/argo => ../../../../../../../pkg/mod/github.com/argoproj/argo@v0.0.0-20200909173032-f930c0296c41
// Windows ..\..\..\..\..\..\..\pkg\mod\github.com\argoproj\argo@v0.0.0-20200909173032-f930c0296c41

replace github.com/kubeflow/pipelines => ../../.. 
// Windows ..\..\..
