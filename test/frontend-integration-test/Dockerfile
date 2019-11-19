FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs:v20190618-0.1.21-79-g2d40ae68-dirty-9d8d84
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
