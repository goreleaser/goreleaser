FROM golang:1.14-alpine

RUN apk add --no-cache bash \
                       bzr \
                       curl \
                       docker-cli \
                       git \
                       mercurial

ENTRYPOINT ["/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser /bin/goreleaser
