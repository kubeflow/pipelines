# How to release third party images

## How to build
```
cd $KFP_SRC
gcloud builds submit --config third_party/argo/cloudbuild.yaml . --substitutions=TAG_NAME="RELEASE_TAG_NAME_HERE"
gcloud builds submit --config third_party/minio/cloudbuild.yaml . --substitutions=TAG_NAME="RELEASE_TAG_NAME_HERE"
```

or you can build locally using docker too like the following
```
cd $KFP_SRC
docker build -t $IMAGE_NAME:$TAG -f third_party/minio/Dockerfile .
```

## Verify your built images are good
Run the following command to start an interactive shell in a new container of the image (the image must have shell installed to be able to run it)
```
docker run -it --entrypoint sh $IMAGE_NAME
```
Or if the image doesn't have a complete OS (like argoproj/workflow-controller)
```
docker save nginx > nginx.tar
tar -xvf nginx.tar
```
This saves layers of the image to a tarball that you can extract and see.

Credits to: https://stackoverflow.com/questions/44769315/how-to-see-docker-image-contents

## Release to gcr.io/ml-pipeline

1. First build images in your own project
2. Use [this gcloud command](https://cloud.google.com/container-registry/docs/managing#tagging_images) to retag your images to gcr.io/ml-pipeline
3. When choosing the new tag, use the same text as the original release tag of the third party image
