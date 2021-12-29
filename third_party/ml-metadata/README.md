# Machine Learning Metadata

Upstream repo location: <https://github.com/google/ml-metadata>.

## Upgrade MLMD versions

First change the MLMD version in VERSION file in current folder, for example: $VERSION=1.2.0.

```bash
echo -n "$VERSION" > VERSION
```

Run the `update_version.sh` script in current folder to update related image references:

```bash
./update_version.sh
```

Run update_dependencies.sh in the following way:

```bash
../../hack/update-all-requirements.sh
```

Make sure the generated files are as expected. Update clients as described below:

### Build golang gRPC client from proto

#### Prerequisites

Make sure you have installed tools and packages in [grpc golang prerequisites](https://grpc.io/docs/languages/go/quickstart/#prerequisites).

#### Command

```bash
make
```

#### Adopt new golang client in backend

After submitting the changes from above command, find all the modules with `github.com/kubeflow/pipelines/third_party/ml-metadata` in all `go.mod` files. Update them by the following steps:

1. Navigate to the folder which contains the `go.mod` that you want to update.
1. Run `go get github.com/kubeflow/pipelines/third_party/ml-metadata@latest`.
1. `go mod tidy`

To learn more, refer to [Upgrading or downgrading a dependency](https://go.dev/doc/modules/managing-dependencies#upgrading). 

### Build JS client from proto

Refer to [frontend/README.md](frontend/README.md) for [MLMD components - Building generated metadata Protocol Bufers](https://github.com/kubeflow/pipelines/blob/master/frontend/README.md#mlmd-components), for example: you can search for `npm run build:replace` command. 
