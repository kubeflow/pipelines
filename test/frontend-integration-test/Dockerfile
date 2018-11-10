FROM gcr.io/ml-pipeline-test/selenium-standalone-chrome-nodejs:v20181109-0.1.1-56-gb190e72-dirty-28e8ac
#To build this image: cd selenium-standalone-chrome-nodejs.Docker && make push

WORKDIR /src

COPY . . 

ENTRYPOINT [ "./run_test.sh" ]
