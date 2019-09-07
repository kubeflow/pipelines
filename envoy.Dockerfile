FROM envoyproxy/envoy:latest

COPY envoy.yaml /etc/envoy.yaml

RUN apt-get update && \
    apt-get install gettext -y

ENTRYPOINT ["/usr/local/bin/envoy", "-c"]
CMD ["/etc/envoy.yaml"]
