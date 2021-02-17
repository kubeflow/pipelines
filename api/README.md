# Pipeline Spec

## Generate golang proto code

Documentation: <https://developers.google.com/protocol-buffers/docs/reference/go-generated>

Download `protoc` compiler binary from: <https://github.com/protocolbuffers/protobuf/releases/tag/v3.14.0>.

Install proto code generator:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go
```

Generate golang proto code:

```bash
make gen_proto
```
