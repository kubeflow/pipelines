module github.com/kubeflow/pipelines

go 1.17

require (
	github.com/Masterminds/squirrel v0.0.0-20190107164353-fa735ea14f09
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/argoproj/argo-workflows/v3 v3.2.3
	github.com/aws/aws-sdk-go v1.36.1
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/eapache/go-resiliency v1.2.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/errors v0.19.9
	github.com/go-openapi/runtime v0.19.24
	github.com/go-openapi/strfmt v0.19.11
	github.com/go-openapi/swag v0.19.13
	github.com/go-openapi/validate v0.20.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/glog v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/google/addlicense v0.0.0-20200906110928-a0294312aa76
	github.com/google/cel-go v0.9.0
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/jinzhu/gorm v1.9.1
	github.com/kubeflow/pipelines/api v0.0.0-20220118175555-e78ed557ddcb
	github.com/kubeflow/pipelines/third_party/ml-metadata v0.0.0-20220118175555-e78ed557ddcb
	github.com/lestrrat-go/strftime v1.0.4
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/peterhellberg/duration v0.0.0-20191119133758-ec6baeebcd10
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	gocloud.dev v0.22.0
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	google.golang.org/genproto v0.0.0-20211221231510-d629cc9a93d5
	google.golang.org/grpc v1.43.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.5
	k8s.io/apimachinery v0.21.5
	k8s.io/client-go v0.21.5
	k8s.io/code-generator v0.21.5
	k8s.io/kubernetes v0.17.9
	sigs.k8s.io/controller-runtime v0.9.7
)

require (
	cloud.google.com/go v0.81.0 // indirect
	cloud.google.com/go/storage v1.12.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210826220005-b48c857c3a0e // indirect
	github.com/antonmedv/expr v1.8.9 // indirect
	github.com/argoproj/pkg v0.11.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/colinmarc/hdfs v1.1.4-0.20180805212432-9746310a4d31 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/doublerebel/bellows v0.0.0-20160303004610-f177d92a03d3 // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/go-ini/ini v1.51.1 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-openapi/analysis v0.19.16 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/loads v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.2 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/wire v0.4.0 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a // indirect
	github.com/jinzhu/now v0.0.0-20181116074157-8ec929ed50c3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/klauspost/compress v1.13.1 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852 // indirect
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.1.0 // indirect
	go.mongodb.org/mongo-driver v1.4.4 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.5 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/api v0.44.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0 // indirect
	gopkg.in/jcmturner/rpc.v0 v0.0.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/apiextensions-apiserver v0.21.4 // indirect
	k8s.io/component-base v0.21.4 // indirect
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027 // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7 // indirect
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	k8s.io/kubernetes => k8s.io/kubernetes v1.11.1
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9
)
