# Pull syft, cosign, and docker-buildx from their upstream images so we
# control the dependency versions (Alpine packages bundle older grpc, see
# CVE-2026-33186) and Dependabot can keep these tags up to date.
FROM anchore/syft:v1.42.4@sha256:e9f29bec38cc856bfd3a7966d2f99711b5b244a531bf121da9de3b47789eecfa AS syft
FROM gcr.io/projectsigstore/cosign:v3.0.6@sha256:de9c65609e6bde17e6b48de485ee788407c9502fa08b8f4459f595b21f56cd00 AS cosign
FROM docker/buildx-bin:0.33.0@sha256:450be95fa632a3986797cd23b8b5d8d5fff47e9fd8e1fa483c9d44b07da2a559 AS buildx

FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166

ARG TARGETPLATFORM

RUN apk add --no-cache bash \
	build-base \
	curl \
	docker-cli \
	git \
	git-lfs \
	gpg \
	mercurial \
	make \
	openssh-client \
	tini \
	upx

COPY --from=syft   /syft          /usr/bin/syft
COPY --from=cosign /ko-app/cosign /usr/bin/cosign
COPY --from=buildx /buildx        /usr/libexec/docker/cli-plugins/docker-buildx

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY $TARGETPLATFORM/goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
