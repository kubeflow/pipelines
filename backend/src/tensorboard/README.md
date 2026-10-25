# TensorBoard viewer runtime

The default KFP viewer image runs TensorBoard without TensorFlow. TensorBoard's
standalone filesystem implementation reads local event files, S3 through boto3,
and GCS through gcsfs. `S3_ENDPOINT`, AWS credentials, and Google application
default credentials retain their normal storage-client meanings. The viewer
controller supplies the log directory, scoped URL prefix, and bind address.

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
