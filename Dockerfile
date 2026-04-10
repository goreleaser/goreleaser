FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166

ARG TARGETPLATFORM

RUN apk add --no-cache bash \
	build-base \
	curl \
	cosign \
	docker-cli \
	docker-cli-buildx \
	git \
	git-lfs \
	gpg \
	mercurial \
	make \
	openssh-client \
	syft \
	tini \
	upx

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY $TARGETPLATFORM/goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
