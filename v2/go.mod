module github.com/kubeflow/pipelines/v2

go 1.15

require (
	github.com/argoproj/argo-workflows/v3 v3.1.1
	github.com/aws/aws-sdk-go v1.36.1
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang/glog v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.9.0
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/kubeflow/pipelines/api v0.0.0-20211020193552-20f28631517d
	github.com/kubeflow/pipelines/third_party/ml-metadata v0.0.0-20211127100422-83295a2f85e9
	github.com/stretchr/testify v1.7.0
	gocloud.dev v0.22.0
	google.golang.org/genproto v0.0.0-20211026145609-4688e4c4e024
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.20.4
)

replace github.com/kubeflow/pipelines/api => ../api
