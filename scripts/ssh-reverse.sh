#!/usr/bin/env bash
# From a host that can reach both the acahti gRPC port and a far runner,
# forward :9000 so the runner agent can use SERVER=127.0.0.1:9000.
#   RUNNER=user@192.0.2.50 ACAHTI_GRPC=192.0.2.10:9000 bash scripts/ssh-reverse.sh
set -euo pipefail
RUNNER="${RUNNER:?set RUNNER=user@host}"
ACAHTI_GRPC="${ACAHTI_GRPC:?set ACAHTI_GRPC=host:9000}"
exec autossh -M 0 -N -o BatchMode=yes -o ServerAliveInterval=30 \
  -R 127.0.0.1:9000:"${ACAHTI_GRPC}" "${RUNNER}"
