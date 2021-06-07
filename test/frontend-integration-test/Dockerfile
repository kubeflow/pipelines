FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs:v20200210-0.2.2-30-g05865480-e3b0c4
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
