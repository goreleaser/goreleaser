#!/bin/bash
set -e
if [ "${1: -4}" == ".deb" ] || [ "${1: -4}" == ".rpm" ]; then
	cd dist
	echo "uploading $1"
	curl -f -q -s -F package="@$1" "https://$FURY_TOKEN@push.fury.io/goreleaser/" >/dev/null
fi
