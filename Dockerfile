# Pull syft, cosign, docker, and docker-buildx from their upstream images so
# we control the dependency versions.
FROM anchore/syft:v1.57.1@sha256:03215d0e98b661d0ecbc8575dbc14d65be1743b5de5df6410aecb7e80f95b060 AS syft
FROM gcr.io/projectsigstore/cosign:v3.1.4@sha256:0f06154836d2de34f887a5c273506d52ca43edaa1d9e53e6510fa78a48f69c2d AS cosign
FROM docker:29.5.3-cli-alpine3.23@sha256:873de13208aab9c1de73fe984fd45883e01464fcfcc85efa20aa56a9ccfe7aa6 AS docker
FROM docker/buildx-bin:v0.38@sha256:bb58c7c322142f342f343755099dffb426919b9bd588ad5f3c607c96f18b77c8 AS buildx

FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf

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
