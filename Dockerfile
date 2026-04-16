FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166

ARG TARGETPLATFORM

# Pinned tool versions — update these to pull newer upstream releases.
ARG SYFT_VERSION=1.42.4
ARG COSIGN_VERSION=3.0.6
ARG BUILDX_VERSION=0.33.0

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

# Install syft, cosign, and docker-buildx from upstream releases to control
# dependency versions and avoid CVEs in Alpine-packaged binaries.
RUN set -eux; \
	case "${TARGETPLATFORM}" in \
		linux/amd64) ARCH=amd64 ;; \
		linux/arm64) ARCH=arm64 ;; \
		*) echo "unsupported platform: ${TARGETPLATFORM}" >&2; exit 1 ;; \
	esac; \
	# syft
	curl -fsSL "https://github.com/anchore/syft/releases/download/v${SYFT_VERSION}/syft_${SYFT_VERSION}_linux_${ARCH}.tar.gz" \
		| tar xz -C /usr/bin syft; \
	# cosign
	curl -fsSL -o /usr/bin/cosign \
		"https://github.com/sigstore/cosign/releases/download/v${COSIGN_VERSION}/cosign-linux-${ARCH}"; \
	chmod +x /usr/bin/cosign; \
	# docker-buildx
	mkdir -p /usr/libexec/docker/cli-plugins; \
	curl -fsSL -o /usr/libexec/docker/cli-plugins/docker-buildx \
		"https://github.com/docker/buildx/releases/download/v${BUILDX_VERSION}/buildx-v${BUILDX_VERSION}.linux-${ARCH}"; \
	chmod +x /usr/libexec/docker/cli-plugins/docker-buildx

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY $TARGETPLATFORM/goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
