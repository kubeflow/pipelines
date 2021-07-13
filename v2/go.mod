module github.com/kubeflow/pipelines/v2

go 1.15

require (
	github.com/aws/aws-sdk-go v1.36.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.5.0
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/kubeflow/pipelines/api v0.0.0
	github.com/stretchr/testify v1.6.1
	gocloud.dev v0.22.0
	google.golang.org/genproto v0.0.0-20201203001206-6486ece9c497
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
)

replace github.com/kubeflow/pipelines/api => ../api
