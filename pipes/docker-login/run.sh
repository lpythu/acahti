#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input REGISTRY
registry="$(input REGISTRY)"
host="$(registry_host "$registry")"
if [[ -z "$host" ]]; then
	echo "error: with.registry must be a hostname (optional http:// or https:// prefix)" >&2
	exit 1
fi
user="$(input USERNAME)"
user="${DOCKER_USERNAME:-$user}"
pass="${DOCKER_PASSWORD:-}"
if [[ -z "$user" || -z "$pass" ]]; then
	echo "error: with.username (or DOCKER_USERNAME) and DOCKER_PASSWORD secret are required" >&2
	exit 1
fi

# ACR Enterprise rejects a static instance password. Official login is
# GetAuthorizationToken; YAML still names acr_username / acr_password, which
# are an Aliyun AccessKey used only to mint the short-lived docker pair.
if [[ "$host" == *.cr.aliyuncs.com ]]; then
	pair="$(
		ACR_HOST="$host" ALIBABA_CLOUD_ACCESS_KEY_ID="$user" ALIBABA_CLOUD_ACCESS_KEY_SECRET="$pass" \
			python3 - <<'PY'
import hmac, hashlib, base64, json, os, time, uuid, urllib.parse, urllib.request

ak = os.environ["ALIBABA_CLOUD_ACCESS_KEY_ID"]
sk = os.environ["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]
host = os.environ["ACR_HOST"]
# saidc-bj-registry.cn-beijing.cr.aliyuncs.com
parts = host.split(".")
if len(parts) < 4 or parts[-3] != "cr":
    raise SystemExit("error: cannot parse ACR region from " + host)
region = parts[-4]


def rpc(action, extra):
    params = {
        "Format": "JSON",
        "Version": "2018-12-01",
        "AccessKeyId": ak,
        "SignatureMethod": "HMAC-SHA1",
        "Timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "SignatureVersion": "1.0",
        "SignatureNonce": uuid.uuid4().hex,
        "Action": action,
        **extra,
    }
    items = sorted((urllib.parse.quote(k, safe="~"), urllib.parse.quote(str(v), safe="~")) for k, v in params.items())
    canonical = "&".join(f"{k}={v}" for k, v in items)
    string_to_sign = "GET&%2F&" + urllib.parse.quote(canonical, safe="~")
    sig = base64.b64encode(hmac.new((sk + "&").encode(), string_to_sign.encode(), hashlib.sha1).digest()).decode()
    params["Signature"] = sig
    url = "https://cr." + region + ".aliyuncs.com/?" + urllib.parse.urlencode(params)
    with urllib.request.urlopen(url, timeout=20) as resp:
        return json.load(resp)


listing = rpc("ListInstance", {})
instances = listing.get("Instances") or []
instance_id = ""
hint = host.split(".")[0]
for inst in instances:
    name = (inst.get("InstanceName") or "").lower()
    if name and name in hint:
        instance_id = inst.get("InstanceId") or ""
        break
if not instance_id and len(instances) == 1:
    instance_id = instances[0].get("InstanceId") or ""
if not instance_id:
    raise SystemExit("error: no ACR instance in " + region + " matching " + host)
tok = rpc("GetAuthorizationToken", {"InstanceId": instance_id})
user = tok.get("TempUsername") or ""
password = tok.get("AuthorizationToken") or ""
if not user or not password:
    raise SystemExit("error: GetAuthorizationToken returned no docker credentials")
print(user + "\n" + password)
PY
	)"
	user="${pair%%$'\n'*}"
	pass="${pair#*$'\n'}"
	if [[ -z "$user" || -z "$pass" || "$user" == "$pass" ]]; then
		echo "error: ACR GetAuthorizationToken did not return a docker user/password" >&2
		exit 1
	fi
fi

printf '%s\n' "$pass" | docker login "$host" -u "$user" --password-stdin >/dev/null
if truthy "$(input HTTP)" || [[ "$registry" == http://* ]]; then
	ensure_registry_http "$host"
fi
echo "OK docker-login ${host}"
