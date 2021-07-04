FROM golang:1.16-alpine

RUN apk add --no-cache bash \
                       curl \
                       docker-cli \
                       git \
                       mercurial \
                       make \
                       build-base

ENTRYPOINT ["/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --allow-untrusted /tmp/goreleaser_*.apk

RUN mkdir -p $HOME/.docker/cli-plugins/ && \
    wget -O $HOME/.docker/cli-plugins/docker-buildx https://github.com/docker/buildx/releases/download/v0.4.1/buildx-v0.4.1.linux-amd64 && \
    chmod a+x $HOME/.docker/cli-plugins/docker-buildx 
