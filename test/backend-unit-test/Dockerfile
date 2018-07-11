# This image has the script to kick off the ML pipeline backend unit test,
# and upload the result to GCS

FROM golang:1.10

RUN curl https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz > /tmp/google-cloud-sdk.tar.gz
RUN mkdir -p /usr/local/gcloud
RUN tar -C /usr/local/gcloud -xvf /tmp/google-cloud-sdk.tar.gz
RUN /usr/local/gcloud/google-cloud-sdk/install.sh
ENV PATH $PATH:/usr/local/gcloud/google-cloud-sdk/bin

# install go-junit-report. It converts go test result to junit xml.
RUN go get -u github.com/jstemmer/go-junit-report
RUN go build github.com/jstemmer/go-junit-report

COPY ./run_test.sh /run_test.sh
RUN chmod +x /run_test.sh

ENTRYPOINT ["/run_test.sh"]
