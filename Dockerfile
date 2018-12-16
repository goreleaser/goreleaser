FROM golang:1.11-alpine

RUN apk add --no-cache bash \
                       curl \
                       docker \
                       git \
                       mercurial \
                       rpm

ENTRYPOINT ["/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser /bin/goreleaser
