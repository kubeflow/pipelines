FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs:v20190507-0.1.19-19-g0b8b4dd7-dirty-20a6f9
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
