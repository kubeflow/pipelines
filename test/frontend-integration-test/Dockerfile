FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs:v20230322-2.0.0b11-144-g87e1c5fcf-dirty-2ca4f7
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
