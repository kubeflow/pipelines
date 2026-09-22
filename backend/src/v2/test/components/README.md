# Native v2 test components

The `.py` files define native `dsl.container_component` components. Their `.yaml`
files are compiler-generated PipelineSpec IR and can be loaded with the v2-only
`kfp.components.load_component_from_file` API.

Regenerate with the current SDK and pipeline-spec package installed in an isolated
Python environment, from the repository root:

```sh
/path/to/venv/bin/python backend/src/v2/test/components/download_gcs_tgz.py
/path/to/venv/bin/python backend/src/v2/test/components/kaniko.py
/path/to/venv/bin/python backend/src/v2/test/components/run_sample.py
/path/to/venv/bin/python -m pytest backend/src/v2/test/components/components_test.py
```

The shell commands, images, normalized input/output names, and defaults are
preserved. URI/path values passed as command-line strings are native `str`
parameters. Compiler binaries, downloaded folders, optional Kaniko context, and
Kaniko's digest file use `system.Artifact` paths. Kaniko can use either a URI
context or the optional context artifact.

`run_sample` installs the checkout's native SDK and `fire` using
`backend/src/v2/test/requirements.txt`, from that directory so the editable SDK
path resolves correctly. This matches the test image's Dockerfile and does not
require the removed deprecated SDK requirements bundle.

The tests compile and reload all three IR files, compose both Kaniko context
variants, and execute the sample runner shell with stubbed tools. They do not
start containers, invoke Kaniko/gsutil, install packages, or contact a cluster.
