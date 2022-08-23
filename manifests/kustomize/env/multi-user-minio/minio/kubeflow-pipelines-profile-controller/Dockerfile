FROM docker.io/minio/mc:RELEASE.2019-08-14T20-49-49Z as minio-cli

FROM python:3.7
# curl -o mc 'https://dl.min.io/client/mc/release/linux-amd64/archive/mc.RELEASE.2019-08-14T20-49-49Z'
COPY --from=minio-cli /usr/bin/mc /app/mc
RUN pip3 install 'boto3==1.22.*' 'minio==7.1.*' 'kubernetes==23.3.*'

ENTRYPOINT ["python3" ]
