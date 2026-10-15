# Machine Learning Metadata

Upstream repo location: <https://github.com/google/ml-metadata>.

## Upgrade MLMD versions

First change the MLMD version in VERSION file in current folder, for example: $VERSION=1.2.0.

```bash
echo -n "$VERSION" > VERSION
```

This directory retains MLMD protobuf bindings for decoding historical 2.18 archives.
The legacy metadata server, metadata writer, and frontend clients are no longer
part of master; this update does not change their former images or requirements.

Install the protobuf tools below, then run from any directory:

```bash
/path/to/pipelines/third_party/ml-metadata/update_version.sh
```

The script runs `make update`: it downloads the sources selected by `VERSION`
and regenerates the vendored Go bindings. Review the source and generated diff,
then run `go test ./backend/src/apiserver/history` from the repository root to
verify supported archives still decode. The root module uses this directory
through a local replacement; no remote module update is needed.

### Build golang gRPC client from proto

#### Prerequisites

Make sure you have installed tools and packages in [grpc golang prerequisites](https://grpc.io/docs/languages/go/quickstart/#prerequisites).

NOTE: The versions for tools are important, following is a record for when the version combination works successfully.

```bash
apt install -y protobuf-compiler=3.15.8
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
```

#### Command

```bash
make
```
