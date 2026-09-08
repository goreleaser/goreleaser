# Pull syft, cosign, docker, and docker-buildx from their upstream images so
# we control the dependency versions.
FROM anchore/syft:v1.51.1@sha256:95fe0835e5bebc6f8b1f8acef68d47d63d594ef4c0f25c097ff853b23cbac74c AS syft
FROM gcr.io/projectsigstore/cosign:v3.1.3@sha256:9e5c2f2edc34351160407ca3416c61855bdf9403c3c5936e0f0be7fc261611b8 AS cosign
FROM docker:29.5.3-cli-alpine3.23@sha256:873de13208aab9c1de73fe984fd45883e01464fcfcc85efa20aa56a9ccfe7aa6 AS docker
FROM docker/buildx-bin:0.37.0@sha256:1387e0aa2b756894e8b6ee10d0f80904cb0badc4356a29e7843b230d0b8fd48f AS buildx

FROM golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125

ARG TARGETPLATFORM

RUN apk add --no-cache bash \
	build-base \
	curl \
	git \
	git-lfs \
	gpg \
	gpg-agent \
	mercurial \
	make \
	openssh-client \
	tini \
	upx

COPY --from=syft   /syft                  /usr/bin/syft
COPY --from=cosign /ko-app/cosign         /usr/bin/cosign
COPY --from=docker /usr/local/bin/docker  /usr/bin/docker
COPY --from=buildx /buildx                /usr/libexec/docker/cli-plugins/docker-buildx

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY $TARGETPLATFORM/goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
