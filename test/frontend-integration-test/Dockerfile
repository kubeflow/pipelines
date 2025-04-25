FROM ghcr.io/kubeflow/kfp-selenium-standalone-chrome-gcloud-nodejs:main-e21bbbaf225dc0d5470df55d6d7254b7e2dd066c
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
