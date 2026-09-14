#!/bin/sh
# Bind-mount ${ACAHTI_DATA} is often root-owned on the host. Chown then drop to the
# gateway uid so invites.json / oauth.json are writable. Process never stays root.
set -eu
dir="${ACAHTI_DATA:-/var/lib/acahti}"
mkdir -p "$dir"
chown -R 65532:65532 "$dir"
exec su-exec 65532:65532 /usr/local/bin/acahti-gateway
