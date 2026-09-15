#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input ENDPOINT
require_input BUCKET
require_input PREFIX
require_input FILES
command -v ossutil >/dev/null || {
	echo "error: ossutil required on the Runner" >&2
	exit 1
}
ak="${OSS_ACCESS_KEY_ID:-}"
sk="${OSS_ACCESS_KEY_SECRET:-}"
if [[ -z "$ak" || -z "$sk" ]]; then
	echo "error: OSS_ACCESS_KEY_ID and OSS_ACCESS_KEY_SECRET secrets are required" >&2
	exit 1
fi
endpoint="$(input ENDPOINT)"
bucket="$(input BUCKET)"
prefix="$(input PREFIX)"
prefix="${prefix#oss://}"
prefix="${prefix#${bucket}/}"
dest="oss://${bucket}/${prefix}"
dest="${dest%/}/"
while IFS= read -r f; do
	[[ -z "$f" ]] && continue
	if [[ "$f" != /* ]]; then
		f="${ROOT}/${f}"
	fi
	if [[ ! -f "$f" ]]; then
		echo "error: file not found: ${f}" >&2
		exit 1
	fi
	echo "==> oss-put ${f} → ${dest}"
	ossutil cp -f "$f" "$dest" --endpoint "$endpoint" -i "$ak" -k "$sk"
done < <(each_line "$(input FILES)")
echo "OK oss-put ${dest}"
