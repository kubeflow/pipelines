# This image has the script to kick off the ML pipeline backend unit test,
# and upload the result to GCS

FROM alpine:3.7

RUN apk add --no-cache curl

COPY ./wait.sh /wait.sh
RUN chmod +x /wait.sh

ENTRYPOINT ["/wait.sh"]
