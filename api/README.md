# Pipeline Spec

## Generate golang proto code

Generate golang proto code:

```bash
make clean-go golang
```

## Generate Python proto package

Generate the SDK-owned `kfp.pipeline_spec` bindings:

```bash
make clean-python python
```

The output is `sdk/python/kfp/pipeline_spec/pipeline_spec_pb2.py` and must be
committed after schema changes. There is no separate pipeline-spec distribution
or version. From the repository root, `make -C sdk python` runs all Python
generators before building the unified SDK.

## Generate both Python and golang proto code

Generate both Python and golang proto:

```bash
make clean all
```

Note, there are no prerequisites, because the generation uses a prebuilt docker image with all the tools necessary.

Documentation: <https://developers.google.com/protocol-buffers/docs/reference/go-generated>
