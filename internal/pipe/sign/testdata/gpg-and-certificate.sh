#!/bin/sh
# A signer that also writes a certificate, which gpg cannot do on its own.
# Every argument is passed to gpg, and the certificate is written to the path
# the sign pipe asked for.
set -e
gpg "$@"
echo "certificate for ${artifact}" >"${certificate}"
