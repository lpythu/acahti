#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input URLS
timeout="$(input TIMEOUT)"
timeout="${timeout:-180}"
ua="$(input USER_AGENT)"
ua="${ua:-acahti-pipe/wait-http}"
deadline=$((SECONDS + timeout))

while IFS= read -r url; do
	[[ -z "$url" ]] && continue
	echo "==> wait ${url}"
	code="000"
	while ((SECONDS < deadline)); do
		code="$(curl -sS -o /dev/null -w '%{http_code}' -A "$ua" --max-time 10 "$url" || true)"
		if [[ "$code" != "502" && "$code" != "503" && "$code" != "504" && "$code" != "000" ]]; then
			echo "OK ${url} ${code}"
			continue 2
		fi
		sleep 3
	done
	echo "error: ${url} still ${code:-000} after ${timeout}s" >&2
	exit 1
done < <(each_item "$(input URLS)")
