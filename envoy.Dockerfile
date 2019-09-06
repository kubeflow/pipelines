FROM envoyproxy/envoy:latest

COPY envoy.yaml /etc/envoy.yaml
COPY envoy-entrypoint.sh /

RUN chmod 500 /envoy-entrypoint.sh

RUN apt-get update && \
    apt-get install gettext -y

ENTRYPOINT ["/envoy-entrypoint.sh"]
