FROM golang:1.18.1-alpine@sha256:1a80c4c090ad01efb2952b0ba2a8cb5aed157eb10963c8c088e443911ac48a88

RUN apk add --no-cache bash \
	curl \
	docker-cli \
	docker-cli-buildx \
	git \
	gpg \
	mercurial \
	make \
	build-base \
	tini

# install cosign
COPY --from=gcr.io/projectsigstore/cosign:v1.7.2@sha256:ad2985a87622d5934a4bc06a61faadff772e377937e42519af4f506e1b019d1e /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
