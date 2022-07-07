#!/bin/bash
set -e
if [ "${1: -4}" == ".deb" ] || [ "${1: -4}" == ".rpm" ]; then
	cd dist
	echo "uploading $1"
	status="$(curl -s -q -o /dev/null -w "%{http_code}" -F package="@$1" "https://$FURY_TOKEN@push.fury.io/goreleaser/")"
	echo "got: $status"
	if [ "$status" == "200" ] || [ "$status" == "409" ]; then
		exit 0
	fi
	exit 1
fi
