FROM golang:1.23.3-alpine@sha256:09742590377387b931261cbeb72ce56da1b0d750a27379f7385245b2b058b63a

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
	tini \
	upx

# install cosign
COPY --from=ghcr.io/sigstore/cosign/cosign:v2.4.0@sha256:9d50ceb15f023eda8f58032849eedc0216236d2e2f4cfe1cdf97c00ae7798cfe /ko-app/cosign /usr/bin/cosign

# install syft
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/v0.84.1/install.sh | sh -s -- -b /usr/local/bin

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
