# deploy stage
FROM ubuntu:20.04

RUN apt-get -y update \
 && apt-get -y install \
    ca-certificates \
    curl \
 && rm -rf /var/lib/apt/lists/*

ARG VERSION=2021.9.3-discovery
ARG SERVER_VERSION=${VERSION}
ARG CORTEZA_SERVER_PATH=https://releases.cortezaproject.org/files/corteza-discovery-searcher-${SERVER_VERSION}-linux-amd64.tar.gz
RUN mkdir /tmp/server
ADD $CORTEZA_SERVER_PATH /tmp/server

VOLUME /data

RUN tar -zxvf "/tmp/server/$(basename $CORTEZA_SERVER_PATH)" -C / && \
    rm -rf "/tmp/server" && \
    mv /corteza-server /corteza

WORKDIR /corteza

#HEALTHCHECK --interval=30s --start-period=1m --timeout=30s --retries=3 \
#    CMD curl --silent --fail --fail-early http://127.0.0.1:80/healthcheck || exit 1

ENV STORAGE_PATH "/data"
ENV PATH "/corteza/bin:${PATH}"
ENV DISCOVERY_SEARCHER_HTTP_ADDR "0.0.0.0:3101"

EXPOSE 3101

ENTRYPOINT ["./bin/corteza-server"]

#CMD ["serve-api"]
