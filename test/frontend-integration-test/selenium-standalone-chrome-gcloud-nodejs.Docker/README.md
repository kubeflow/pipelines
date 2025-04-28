### Build and Push this image to KFP GHCR

1. Login to GHCR container registry: `echo "<PAT>" | docker login ghcr.io -u <USERNAME> --password-stdin`
    * Replace `<PAT>` with a GitHub Personal Access Token (PAT) with the write:packages and `read:packages` scopes, as well as `delete:packages` if needed.
1. Set `TAG=` to `main-<commit-hash>` where `<commit-hash>` is the commit used to build the image on (your current HEAD)/
1. Update the [Dockerfile](`./Dockerfile`) and build the image by running `docker build -t ghcr.io/kubeflow/kfp-selenium-standalone-chrome-gcloud-nodejs:$TAG .`
1. Push the new container by running `docker push ghcr.io/kubeflow/kfp-selenium-standalone-chrome-gcloud-nodejs:$TAG`.
1. Update the version in front end integration [Dockerfile](test/frontend-integration-test/Dockerfile) to point to your new image.
