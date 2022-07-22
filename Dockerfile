FROM golang:1.18.4-alpine@sha256:c9a90742f5457fae80d8f9f1c9fc6acd6884c749dc6c5b11c22976973564dd4f

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
COPY --from=gcr.io/projectsigstore/cosign:v1.10.0@sha256:a719237925984033fb72685c1998d922c903bbe62464f6d401b5108d3195bb94 /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
