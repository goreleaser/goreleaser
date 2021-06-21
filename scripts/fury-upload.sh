#!/bin/bash
set -e
if [ "${1: -4}" == ".deb" ] || [ "${1: -4}" == ".rpm" ]; then
	cd dist
	curl -F package="@$1" "https://$FURY_TOKEN@push.fury.io/goreleaser/"
fi
