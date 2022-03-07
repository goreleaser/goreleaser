FROM golang:1.17.8-alpine

RUN apk add --no-cache bash \
	curl \
	docker-cli \
	docker-cli-buildx \
	git \
	gpg \
	mercurial \
	make \
	build-base \
	tini

# install cosign
COPY --from=gcr.io/projectsigstore/cosign:v1.5.1@sha256:6247b2e693b0e6a62dcfa75eb46b698c1f4cd1aca36aaefafd4bbb2f2b2af717 /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
