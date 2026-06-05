# Pull syft, cosign, docker, and docker-buildx from their upstream images so
# we control the dependency versions.
FROM anchore/syft:v1.45.0@sha256:e9695f12fca6bf93a6e106c8f7001a13d00608b92e8a73930aacc401a8727ac8 AS syft
FROM gcr.io/projectsigstore/cosign:v3.0.6@sha256:de9c65609e6bde17e6b48de485ee788407c9502fa08b8f4459f595b21f56cd00 AS cosign
FROM docker:29.5.3-cli-alpine3.23@sha256:873de13208aab9c1de73fe984fd45883e01464fcfcc85efa20aa56a9ccfe7aa6 AS docker
FROM docker/buildx-bin:0.34.1@sha256:ba49f75261dd3ac85491d370a9c38306454a84c5554be4e67de601cd59847cb6 AS buildx

FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f

ARG TARGETPLATFORM

RUN apk add --no-cache bash \
	build-base \
	curl \
	git \
	git-lfs \
	gpg \
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
