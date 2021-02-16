FROM golang:1.16rc1-alpine3.13

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
