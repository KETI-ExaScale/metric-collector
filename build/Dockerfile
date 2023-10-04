FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/metric-collector \
    USER_UID=root \
    USER_NAME=metric-collector
COPY _output/bin/metric-collector ${OPERATOR}


ENTRYPOINT ["/usr/local/bin/metric-collector"]

USER ${USER_UID}
