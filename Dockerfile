FROM golang:1.17.7-alpine

RUN apk add --no-cache bash \
	curl \
	docker-cli \
	docker-cli-buildx \
	git \
	mercurial \
	make \
	build-base \
	tini

# install cosign
COPY --from=gcr.io/projectsigstore/cosign:v1.4.1@sha256:502d5130431e45f28c51d2c24a05ef5ccd3fd916bcc91db0c8bee3a81e09a0bb /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
