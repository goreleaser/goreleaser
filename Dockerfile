# Pull syft, cosign, docker, and docker-buildx from their upstream images so
# we control the dependency versions.
FROM anchore/syft:v1.44.0@sha256:86fde6445b483d902fe011dd9f68c4987dd94e07da1e9edc004e3c2422650de6 AS syft
FROM gcr.io/projectsigstore/cosign:v3.0.6@sha256:de9c65609e6bde17e6b48de485ee788407c9502fa08b8f4459f595b21f56cd00 AS cosign
FROM docker:29.4.3-cli-alpine3.23@sha256:51e23845f5caff1e688a2fae003b0c69d635c9200ad544731db1593731df1d3a AS docker
FROM docker/buildx-bin:0.34.0@sha256:1023c4a1ac77cee49520b620e1fa29d78be4ee3660bffd1a23b8a35f4e9ca417 AS buildx

FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d

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
