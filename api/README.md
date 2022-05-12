# Pipeline Spec

## Generate golang proto code

Generate golang proto code:

```bash
make clean-go golang
```

## Generate Python proto package

Generate kfp-pipeline-spec:

Update `VERSION` in [kfp_pipeline_spec/python/setup.py](https://github.com/kubeflow/pipelines/blob/master/api/kfp_pipeline_spec/python/setup.py) if applicable.

```bash
make clean-python python
```

## Generate both Python and golang proto code

Generate both Python and golang proto:

```bash
make clean all
```

Note, there are no prerequisites, because the generation uses a prebuilt docker image with all the tools necessary.

Documentation: <https://developers.google.com/protocol-buffers/docs/reference/go-generated>
