FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-gcloud-nodejs:v20190506-0.1.19-18-g96ab0ccc-dirty-e39006
#To build this image: cd selenium-standalone-chrome-gcloud-nodejs.Docker && make push

COPY . /src

WORKDIR /src

ENTRYPOINT [ "./run_test.sh" ]
