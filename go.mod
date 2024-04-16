module github.com/kubeflow/pipelines

require (
	github.com/Masterminds/squirrel v0.0.0-20190107164353-fa735ea14f09
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/argoproj/argo-workflows/v3 v3.3.10
	github.com/aws/aws-sdk-go v1.42.50
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/eapache/go-resiliency v1.2.0
	github.com/emicklei/go-restful v2.16.0+incompatible // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/fsnotify/fsnotify v1.5.1
	github.com/go-openapi/errors v0.20.2
	github.com/go-openapi/runtime v0.21.1
	github.com/go-openapi/strfmt v0.21.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.3
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/glog v1.1.0
	github.com/golang/protobuf v1.5.3
	github.com/google/addlicense v0.0.0-20200906110928-a0294312aa76
	github.com/google/cel-go v0.9.0
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/jackc/pgx/v5 v5.4.2
	github.com/jinzhu/gorm v1.9.1
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/kubeflow/pipelines/api v0.0.0-20230331215358-758c91f76784
	github.com/kubeflow/pipelines/kubernetes_platform v0.0.0-20240403164522-8b2a099e8c9f
	github.com/kubeflow/pipelines/third_party/ml-metadata v0.0.0-20230810215105-e1f0c010f800
	github.com/lestrrat-go/strftime v1.0.4
	github.com/mattn/go-sqlite3 v1.14.18
	github.com/minio/minio-go/v6 v6.0.57
	github.com/peterhellberg/duration v0.0.0-20191119133758-ec6baeebcd10
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/client_model v0.4.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.8.4
	gocloud.dev v0.22.0
	golang.org/x/net v0.19.0
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
	google.golang.org/grpc v1.54.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.24.17
	k8s.io/apimachinery v0.24.17
	k8s.io/client-go v0.24.3
	k8s.io/code-generator v0.23.3
	sigs.k8s.io/controller-runtime v0.11.1
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.9.1
	github.com/go-git/go-git/v5 => github.com/go-git/go-git/v5 v5.11.0
	github.com/go-jose/go-jose/v3 => github.com/go-jose/go-jose/v3 v3.0.1
	google.golang.org/grpc => google.golang.org/grpc v1.56.3
)

go 1.13
