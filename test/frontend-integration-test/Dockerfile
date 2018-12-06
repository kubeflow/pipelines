FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs:v20181204-0.1.3-rc.3-14-g0794834-dirty-8996ed
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY --chown=seluser . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
