FROM golang:1.19.1-alpine@sha256:d475cef843a02575ebdcb1416d98cd76bab90a5ae8bc2cd15f357fc08b6a329f

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
COPY --from=gcr.io/projectsigstore/cosign:v1.11.1@sha256:f9fd5a287a67f4b955d08062a966df10f9a600b6b8583fd367bce3f1f000a429 /ko-app/cosign /usr/local/bin/cosign

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD [ "-h" ]

COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY goreleaser_*.apk /tmp/
RUN apk add --no-cache --allow-untrusted /tmp/goreleaser_*.apk
