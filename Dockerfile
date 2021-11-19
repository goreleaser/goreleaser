FROM golang:1.17.3-alpine

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
COPY --from=gcr.io/projectsigstore/cosign:v1.3.1@sha256:3cd9b3a866579dc2e0cf2fdea547f4c9a27139276cc373165c26842bc594b8bd /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
