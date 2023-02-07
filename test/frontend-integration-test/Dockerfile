FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs:v20230201-2.0.0b11-74-g395719cab-dirty-9bed65
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
