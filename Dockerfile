FROM golang:1.18.2-alpine@sha256:4795c5d21f01e0777707ada02408debe77fe31848be97cf9fa8a1462da78d949

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
COPY --from=gcr.io/projectsigstore/cosign:v1.7.2@sha256:ad2985a87622d5934a4bc06a61faadff772e377937e42519af4f506e1b019d1e /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
