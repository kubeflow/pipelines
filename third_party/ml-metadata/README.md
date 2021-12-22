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

## Build golang gRPC client from proto

### Prerequisites

Make sure you have installed tools and packages in [grpc golang prerequisites](https://grpc.io/docs/languages/go/quickstart/#prerequisites).

### Command

```bash
make
```

## Build JS client from proto

Refer to [frontend/README.md](frontend/README.md) for [MLMD components - Building generated metadata Protocol Bufers](https://github.com/kubeflow/pipelines/blob/master/frontend/README.md#mlmd-components), for example: you can search for `npm run build:replace` command. 
