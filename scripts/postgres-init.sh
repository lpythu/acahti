#!/bin/bash
# First-boot only (empty PG data dir). Image already created POSTGRES_DB=forgejo.
set -euo pipefail
psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER}" --dbname "${POSTGRES_DB}" \
  -c "CREATE DATABASE woodpecker OWNER ${POSTGRES_USER};"
psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER}" --dbname "${POSTGRES_DB}" \
  -c "CREATE DATABASE acahti OWNER ${POSTGRES_USER};"
