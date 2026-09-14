#!/usr/bin/env bash
# buildof: tagPush vX.Y.Z → test, upload OSS, point latest, create office agent_rollout.
set -euo pipefail

: "${ROOT:?set ROOT}"
cd "$ROOT"

normalize_ref() {
	local raw="$1"
	raw="${raw#refs/tags/}"
	raw="${raw#refs/heads/}"
	printf '%s' "$raw"
}

strip_v() {
	printf '%s' "${1#v}"
}

TAG="$(normalize_ref "${CI_COMMIT_TAG:-}")"
if [ -z "$TAG" ]; then
	TAG="$(normalize_ref "${CI_COMMIT_REF_NAME:-}")"
fi
if [ -z "$TAG" ]; then
	TAG="$(git describe --tags --exact-match HEAD 2>/dev/null || true)"
fi
VERSION="$(strip_v "$TAG")"
if ! printf '%s' "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "error: release tag must be vX.Y.Z, got ${TAG:-unset}" >&2
	exit 1
fi
if [ "$TAG" != "v${VERSION}" ]; then
	echo "error: exag tag must be v${VERSION}, got $TAG" >&2
	exit 1
fi

export GOPROXY="${GOPROXY:-https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct}"
export GOPRIVATE="${GOPRIVATE:-go.saidc.ai}"
export GONOSUMDB="${GONOSUMDB:-go.saidc.ai}"
export GOWORK="${GOWORK:-off}"
export GOFLAGS="${GOFLAGS:--mod=vendor}"
export PATH="/usr/local/go/bin:/usr/local/bin:/root:${HOME}/.local/bin:${PATH}"

link_ossutil() {
	local bin="$1"
	local dir
	dir="$(dirname "$bin")"
	PATH="${dir}:${PATH}"
	export PATH
	if [ "$(basename "$bin")" != ossutil ]; then
		ln -sfn "$bin" "${dir}/ossutil"
	fi
}

ossutil_cmd() {
	extra=()
	[ -n "${OSSUTIL_CONFIG:-}" ] && extra+=(-c "${OSSUTIL_CONFIG}")
	[ -n "${OSS_ACCESS_KEY_ID:-}" ] && extra+=(-i "${OSS_ACCESS_KEY_ID}")
	[ -n "${OSS_ACCESS_KEY_SECRET:-}" ] && extra+=(-k "${OSS_ACCESS_KEY_SECRET}")
	command ossutil "$@" "${extra[@]}"
}

load_oss_creds() {
	local f
	for f in \
		"${OSSUTIL_CONFIG:-}" \
		/root/.ossutilconfig \
		"${HOME}/.ossutilconfig" \
		/root/.ossutil/config \
		"${HOME}/.ossutil/config"; do
		if [ -n "$f" ] && [ -f "$f" ]; then
			OSSUTIL_CONFIG="$f"
			export OSSUTIL_CONFIG
			echo "==> ossutil config $f"
			return
		fi
	done
	for f in /root/.harbor/oss.env "${HOME}/.harbor/oss.env"; do
		if [ -f "$f" ]; then
			# shellcheck disable=SC1090
			set -a && source "$f" && set +a
			break
		fi
	done
	if [ -n "${OSS_ACCESS_KEY_ID:-}" ] && [ -n "${OSS_ACCESS_KEY_SECRET:-}" ]; then
		echo "==> ossutil using OSS_ACCESS_KEY_ID from env"
		return
	fi
	for f in /root/.aliyun/config.json "${HOME}/.aliyun/config.json"; do
		if [ -f "$f" ]; then
			eval "$(python3 -c '
import json, shlex, sys
d = json.load(open(sys.argv[1]))
cur = d.get("current") or "default"
for pr in d.get("profiles") or []:
    if (pr.get("name") or "") in (cur, "default"):
        ak, sk = pr.get("access_key_id") or "", pr.get("access_key_secret") or ""
        if ak and sk:
            print("OSS_ACCESS_KEY_ID=" + shlex.quote(ak))
            print("OSS_ACCESS_KEY_SECRET=" + shlex.quote(sk))
            print("export OSS_ACCESS_KEY_ID OSS_ACCESS_KEY_SECRET")
            break
' "$f")"
			if [ -n "${OSS_ACCESS_KEY_ID:-}" ] && [ -n "${OSS_ACCESS_KEY_SECRET:-}" ]; then
				echo "==> ossutil using access key from $f"
				return
			fi
		fi
	done
	echo "error: ossutil has no credentials (looked /root/.ossutilconfig, ~/.ossutilconfig, /root/.harbor/oss.env, ~/.aliyun/config.json)" >&2
	exit 1
}

ensure_ossutil() {
	if command -v ossutil >/dev/null 2>&1; then
		return
	fi
	local c
	for c in /usr/local/bin/ossutil /usr/local/bin/ossutil64 /usr/bin/ossutil /usr/bin/ossutil64 \
		/root/ossutil64 /root/ossutil "${HOME}/ossutil64" "${HOME}/.local/bin/ossutil64" \
		/opt/ossutil/ossutil64; do
		if [ -x "$c" ]; then
			link_ossutil "$c"
			return
		fi
	done
	c="${HOME}/.local/bin/ossutil64"
	mkdir -p "$(dirname "$c")"
	echo "==> ossutil not on PATH; downloading to $c"
	curl -fsSL "https://gosspublic.alicdn.com/ossutil/1.7.18/ossutil64" -o "$c"
	chmod +x "$c"
	link_ossutil "$c"
}

if [ ! -d "$ROOT/vendor" ]; then
	echo "error: vendor/ missing; run go mod vendor before tagging" >&2
	exit 1
fi

go_cmd() {
	if command -v go >/dev/null 2>&1; then
		go "$@"
		return
	fi
	extra=()
	[ -n "${GOOS:-}" ] && extra+=(-e "GOOS=${GOOS}")
	[ -n "${GOARCH:-}" ] && extra+=(-e "GOARCH=${GOARCH}")
	[ -n "${CGO_ENABLED:-}" ] && extra+=(-e "CGO_ENABLED=${CGO_ENABLED}")
	docker run --rm \
		--network="${DOCKER_BUILD_NETWORK:-host}" \
		-e GOPROXY -e GOPRIVATE -e GONOSUMDB -e GOWORK -e GOFLAGS \
		"${extra[@]}" \
		-v "$ROOT":/src -w /src \
		golang:1.25-alpine \
		go "$@"
}

echo "==> exag Go tests"
go_cmd test ./...

DIST="dist/cicd"
rm -rf "$DIST"
mkdir -p "$DIST"
LDFLAGS="-X go.saidc.ai/exag/internal/runtime.Version=${VERSION}"
echo "==> build linux amd64/arm64 version=$VERSION"
# Relative -o so docker (WORKDIR /src) writes onto the mounted tree.
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go_cmd build -ldflags "$LDFLAGS" -o "$DIST/exag-linux-amd64" ./cmd/exag
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go_cmd build -ldflags "$LDFLAGS" -o "$DIST/exag-linux-arm64" ./cmd/exag
(
	cd "$DIST"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum exag-linux-amd64 exag-linux-arm64 >SHA256SUMS
	else
		shasum -a 256 exag-linux-amd64 exag-linux-arm64 >SHA256SUMS
	fi
)

BUCKET="${OSS_BUCKET:-saidc}"
ENDPOINT="${OSS_ENDPOINT:-https://oss-cn-hongkong.aliyuncs.com}"
BIN_PREFIX="oss://${BUCKET}/exag/${VERSION}/"
SCRIPTS_PREFIX="oss://${BUCKET}/exag/scripts/"
ensure_ossutil
load_oss_creds
echo "==> upload $BIN_PREFIX"
ossutil_cmd cp -f "$DIST/exag-linux-amd64" "$BIN_PREFIX" --endpoint "$ENDPOINT"
ossutil_cmd cp -f "$DIST/exag-linux-arm64" "$BIN_PREFIX" --endpoint "$ENDPOINT"
ossutil_cmd cp -f "$ROOT/scripts/install.sh" "$SCRIPTS_PREFIX" --endpoint "$ENDPOINT"
ossutil_cmd cp -f "$ROOT/scripts/uninstall.sh" "$SCRIPTS_PREFIX" --endpoint "$ENDPOINT"

DIST_DIR="$DIST"
# shellcheck source=../scripts/publish-latest.sh
. "$ROOT/scripts/publish-latest.sh"
publish_latest_pointer "$VERSION"

EXHUB_BASE_URL="${EXHUB_BASE_URL:-https://exhub.dev.saidc.ai}"
if [ -z "${EXHUB_API_TOKEN:-}" ]; then
	for f in /root/.harbor/exhub-office.env "${HOME}/.harbor/exhub-office.env"; do
		if [ -f "$f" ]; then
			# shellcheck disable=SC1090
			set -a && source "$f" && set +a
			break
		fi
	done
fi
if [ -z "${EXHUB_API_TOKEN:-}" ]; then
	echo "warning: EXHUB_API_TOKEN unset; skip agent_rollout" >&2
	echo "OK exag published $VERSION (OSS only)"
	exit 0
fi

SHA_AMD64="$(awk '/exag-linux-amd64$/{print $1; exit}' "$DIST/SHA256SUMS")"
SHA_ARM64="$(awk '/exag-linux-arm64$/{print $1; exit}' "$DIST/SHA256SUMS")"
URL_AMD64="https://saidc.oss-cn-hongkong.aliyuncs.com/exag/${VERSION}/exag-linux-amd64"
URL_ARM64="https://saidc.oss-cn-hongkong.aliyuncs.com/exag/${VERSION}/exag-linux-arm64"

VERSION="$VERSION" \
ART_URL_AMD64="$URL_AMD64" ART_SHA_AMD64="$SHA_AMD64" \
ART_URL_ARM64="$URL_ARM64" ART_SHA_ARM64="$SHA_ARM64" \
python3 - <<PY >"$DIST/rollout.json"
import json, os
version = os.environ["VERSION"]
payload = {
  "idempotency_key": f"wf-rollout-{version}-all",
  "kind": "agent_rollout",
  "strategy": {"batch_size": 5, "max_failure_ratio": 0.2},
  "doc": {
    "target_version": version,
    "artifacts": {
      "linux/amd64": {"url": os.environ["ART_URL_AMD64"], "sha256": os.environ["ART_SHA_AMD64"]},
      "linux/arm64": {"url": os.environ["ART_URL_ARM64"], "sha256": os.environ["ART_SHA_ARM64"]},
    },
    "selector": {"all": True},
  },
}
print(json.dumps(payload))
PY

echo "==> POST agent_rollout target=$VERSION"
curl -fsS -A 'express-e2e/1.0' \
	-H "Authorization: Bearer ${EXHUB_API_TOKEN}" \
	-H 'Content-Type: application/json' \
	--data-binary @"$DIST/rollout.json" \
	"${EXHUB_BASE_URL}/ops/workflows" | python3 -m json.tool | head -40
echo "OK exag published $VERSION"
