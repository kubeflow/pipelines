# This image has the script to kick off the ML pipeline initialization test,
# and upload the result to GCS

FROM golang:1.13.15

RUN curl https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz > /tmp/google-cloud-sdk.tar.gz
RUN mkdir -p /usr/local/gcloud
RUN tar -C /usr/local/gcloud -xf /tmp/google-cloud-sdk.tar.gz
RUN /usr/local/gcloud/google-cloud-sdk/install.sh
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

# install go-junit-report. It converts go test result to junit xml.
RUN go get -u github.com/jstemmer/go-junit-report
RUN go build github.com/jstemmer/go-junit-report

COPY . /go/src/github.com/kubeflow/pipelines

ENTRYPOINT ["/go/src/github.com/kubeflow/pipelines/test/initialization-test/run_test.sh"]
