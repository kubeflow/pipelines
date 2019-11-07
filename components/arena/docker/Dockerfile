FROM golang:1.10-stretch as build

RUN mkdir -p /go/src/github.com/kubeflow && \
    cd /go/src/github.com/kubeflow && \
    git clone https://github.com/kubeflow/arena.git

WORKDIR /go/src/github.com/kubeflow/arena

RUN cd /go/src/github.com/kubeflow/arena && make

RUN wget --no-verbose https://storage.googleapis.com/kubernetes-helm/helm-v2.9.1-linux-amd64.tar.gz && \
    tar -xf helm-v2.9.1-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    chmod u+x /usr/local/bin/helm

ENV K8S_VERSION v1.11.2
RUN curl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl && chmod +x /usr/local/bin/kubectl

# FROM python:3.6.8-stretch

FROM python:3.7-alpine3.9 

RUN apk update && \
    apk add --no-cache ca-certificates py-dev python-setuptools wget unzip git bash \
    rm -rf /var/cache/apk/*

RUN pip install --upgrade pip && \
    pip install pyyaml==3.12 six==1.11.0 requests==2.18.4

COPY --from=build /go/src/github.com/kubeflow/arena/bin/arena /usr/local/bin/arena

COPY --from=build /usr/local/bin/helm /usr/local/bin/helm

COPY --from=build /go/src/github.com/kubeflow/arena/kubernetes-artifacts /root/kubernetes-artifacts

COPY --from=build /usr/local/bin/kubectl /usr/local/bin/kubectl

COPY --from=build /go/src/github.com/kubeflow/arena/charts /charts

ENV PYTHONPATH "${PYTHONPATH}:/root"

ADD . /root

WORKDIR /root

ENTRYPOINT ["python","arena_launcher.py"]
