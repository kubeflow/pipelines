# Profile controller runtime

`kfp-profile-controller` provides Python and the pinned `botocore` dependency for
`sync.py`. Kubernetes mounts the controller code at `/hooks` from the existing
ConfigMap. The image runs as UID 1000/group 0, matching the deployment's identity,
and does not include the Kubernetes administration tools from `alpine/k8s`.
The Python slim base still includes its normal shell and package-management tools;
this is not a distroless image.

Build with this directory as the context:

```sh
docker build -t kfp-profile-controller:local .
```

The base tag identifies Python 3.13.18 on Debian 13. Its pinned image-index digest
includes Linux AMD64 and ARM64. Refresh the tag and digest together, then rebuild
and scan both platforms before publication. Do not replace the pinned digest with
a floating tag.

Refresh the runtime dependency lock from this directory:

```sh
uv --no-config pip compile --python 3.13 --generate-hashes --no-emit-index-url \
  --output-file requirements.txt requirements.in
```

Build-time pip verifies every dependency hash and installs wheels only. The
installed distribution metadata is retained for vulnerability scanning.
`requirements.in` contains runtime packages only; pytest, requests and PyYAML are
used by the test suite and are not added to the runtime image.

Validate profile reconciliation with `run_tests.sh`. Image acceptance must also
exercise the existing UID, ConfigMap mount, health endpoint, and SeaweedFS IAM/S3
behavior using the built image; dependency resolution and unit tests alone do not
establish container compatibility.
