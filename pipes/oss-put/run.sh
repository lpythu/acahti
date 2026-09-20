#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input ENDPOINT
require_input BUCKET
require_input PREFIX
require_input FILES
bindir="${HOME}/.local/lib/acahti"
mkdir -p "$bindir"
bin="${bindir}/ossutil"
url="https://gosspublic.alicdn.com/ossutil/1.7.18/ossutil64"
if [[ ! -x "$bin" ]]; then
	echo "==> install ossutil from ${url}"
	curl -fsSL "$url" -o "$bin"
	chmod 755 "$bin"
fi
if [[ ! -x "$bin" ]]; then
	echo "error: ossutil required on the Runner" >&2
	exit 1
fi
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
region="${endpoint#https://}"
region="${region#http://}"
region="${region%%/*}"
region="${region#oss-}"
region="${region%%.aliyuncs.com}"
region="${region%-internal}"
oss_cp() {
	local src="$1"
	extra=()
	if ! "$bin" version >/dev/null 2>&1; then
		extra+=(--region "$region")
	fi
	"$bin" cp -f "$src" "$dest" --endpoint "$endpoint" -i "$ak" -k "$sk" "${extra[@]}"
}
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
	oss_cp "$f"
done < <(each_line "$(input FILES)")
echo "OK oss-put ${dest}"
