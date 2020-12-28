FROM golang:1.15-alpine

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

COPY goreleaser /bin/goreleaser
