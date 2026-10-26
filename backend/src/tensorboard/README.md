# TensorBoard viewer runtime

The default KFP viewer image runs TensorBoard without TensorFlow. Master uses
the published `kfp-tensorboard:master` tag; release stamping replaces it with
the release version in both the controller and UI. TensorBoard's
standalone filesystem implementation reads local event files, S3 through boto3,
and GCS through gcsfs. `S3_ENDPOINT`, AWS credentials, and Google application
default credentials retain their normal storage-client meanings. The viewer
controller supplies the log directory, scoped URL prefix, and bind address.

The image's `tensorboard` command defaults to `--load_fast=false` so the server
uses those Python filesystem clients for event ingestion too. TensorBoard's
experimental Rust loader can otherwise select a separate GCS client with
different credential and endpoint handling. The wrapper forwards all arguments
to the original `tensorboard-cli` command; an explicit later `--load_fast` option
can override the default. Custom viewer images retain their own commands and
loader settings.

The image disables GCS directory-listing caching so viewers discover event files
created or updated by training jobs while the viewer is running. It selects the
established gcsfs reader with `GCSFS_EXPERIMENTAL_ZB_HNS_SUPPORT=false`; standard
GCS buckets are supported without experimental zonal/HNS storage-layout probes.
The filesystem defaults apply to both `gs://` and `gcs://` paths. Acceptance
checks must retain these image defaults and verify growing and rotated event
objects without restarting the viewer.

Scalar, image, and graph event viewers are supported. TensorFlow-dependent
plugins and arbitrary training code are not included. Pipelines that require
additional plugins can continue to supply their own viewer image. Existing
viewer pods are not replaced by changing the default; stop and restart a viewer
to use the new runtime.

The UI offers the maintained default and any image supplied by the pipeline;
it no longer offers historical TensorFlow versions. The legacy server
`tfversion` API and `VIEWER_TENSORBOARD_TF_IMAGE_NAME` override continue to select
a custom TensorFlow repository/tag and do not use the KFP default image.

Build from the repository root:

```sh
docker build -f backend/Dockerfile.tensorboard -t kfp-tensorboard:test .
docker run --rm kfp-tensorboard:test tensorboard --version
```

Regenerate dependency hashes with the command recorded at the top of
`requirements.txt`. Qualify dependency updates against scalar, image, and graph
events, browser navigation under a URL prefix, and actual S3/GCS reads. Installing
TensorFlow in this image would change TensorBoard's filesystem implementation
and requires repeating that storage qualification.

Dependency update scheduling covers `/backend/src/tensorboard` for pip and
`/backend` for Docker base images on the default branch. These weekly
version-update PRs require a fresh image scan and the storage, reload,
authorization, and viewer lifecycle acceptance checks before merging.

KFP maintains this small runtime because current TensorFlow images no longer
include TensorBoard, and the previous images contained obsolete dependencies.
The commitment is the viewer runtime and its local/S3/GCS event readers, not a
TensorFlow training distribution or additional plugins. The runtime is built,
published, scanned, and versioned alongside the other KFP images.
