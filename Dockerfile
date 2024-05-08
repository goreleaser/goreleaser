FROM golang:1.22.3-alpine@sha256:2a882244fb51835ebbd8313bffee83775b0c076aaf56b497b43d8a4c72db65e1

RUN apk add --no-cache bash \
	curl \
	docker-cli \
	docker-cli-buildx \
	git \
	gpg \
	mercurial \
	make \
	openssh-client \
	build-base \
	tini

# install cosign
COPY --from=gcr.io/projectsigstore/cosign:v2.1.1@sha256:411ace177097a33cb2ee74028a87ffdcb70965003cd1378c1ec7bf9f9dec9359 /ko-app/cosign /usr/bin/cosign

# install syft
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/v0.84.1/install.sh | sh -s -- -b /usr/local/bin

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
