#!/usr/bin/env python3
"""Activate every org repo in Woodpecker and pin .acahti/pipelines.

Woodpecker 3 only accepts POST /api/repos?forge_remote_id=<forge id>.
The id is Forgejo's numeric repository id, not owner/name. A 409 body is
plain text ("Repository is already active."), not JSON.
"""

import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request

ORG = os.environ.get("ACAHTI_ORG", "").strip()
FJ = os.environ.get("FORGEJO_LOOPBACK", "http://127.0.0.1:3000").rstrip("/")
WP = os.environ.get("WOODPECKER_LOOPBACK", "http://127.0.0.1:8000").rstrip("/")
FJ_TOKEN = os.environ.get("ACAHTI_ADMIN_TOKEN", "").strip()
WP_TOKEN = os.environ.get("WOODPECKER_TOKEN", "").strip()
CONFIG = ".acahti/pipelines"
CSRF_MARK = 'WOODPECKER_CSRF = "'
_csrf = ""


def parse_csrf(raw: str) -> str:
    i = raw.find(CSRF_MARK)
    if i < 0:
        return ""
    s = raw[i + len(CSRF_MARK) :]
    j = s.find('"')
    if j < 0:
        return ""
    return s[:j]


def decode(raw: bytes) -> object:
    if not raw:
        return None
    text = raw.decode("utf-8", "replace")
    if text.lstrip()[:1] in "{[":
        try:
            return json.loads(text)
        except json.JSONDecodeError:
            return text[:300]
    return text[:300]


def activate_query(remote_id: str) -> str:
    return "/api/repos?forge_remote_id=" + urllib.parse.quote(remote_id, safe="")


def lookup_path(full: str) -> str:
    owner, _, name = full.partition("/")
    return "/api/repos/lookup/" + urllib.parse.quote(owner) + "/" + urllib.parse.quote(name)


def csrf() -> str:
    global _csrf
    if _csrf:
        return _csrf
    req = urllib.request.Request(
        WP + "/ci/web-config.js",
        headers={"Cookie": "user_sess=" + WP_TOKEN},
    )
    try:
        with urllib.request.urlopen(req) as resp:
            _csrf = parse_csrf(resp.read().decode("utf-8", "replace"))
    except urllib.error.HTTPError:
        return ""
    return _csrf


def fj(path: str) -> object:
    req = urllib.request.Request(
        FJ + path,
        headers={"Authorization": "token " + FJ_TOKEN},
    )
    with urllib.request.urlopen(req) as resp:
        return json.load(resp)


def wp(method: str, path: str, data: dict | None = None) -> tuple[int, object]:
    body = None if data is None else json.dumps(data).encode()
    headers = {
        "Authorization": "Bearer " + WP_TOKEN,
        "Cookie": "user_sess=" + WP_TOKEN,
    }
    if body is not None:
        headers["Content-Type"] = "application/json"
    if method not in ("GET", "HEAD"):
        token = csrf()
        if token:
            headers["X-CSRF-TOKEN"] = token
    req = urllib.request.Request(
        WP + "/ci" + path,
        data=body,
        headers=headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(req) as resp:
            return resp.status, decode(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, decode(e.read()) or e.reason


def org_repos() -> list[tuple[str, str]]:
    page = 1
    out: list[tuple[str, str]] = []
    while True:
        rows = fj(f"/api/v1/orgs/{urllib.parse.quote(ORG)}/repos?page={page}&limit=50")
        if not isinstance(rows, list) or not rows:
            break
        for row in rows:
            if not isinstance(row, dict):
                continue
            full = str(row.get("full_name") or "")
            remote = row.get("id")
            if full and remote is not None:
                out.append((full, str(remote)))
        if len(rows) < 50:
            break
        page += 1
    return out


def repo_id(body: object) -> str:
    if isinstance(body, dict) and body.get("id"):
        return str(body["id"])
    return ""


def woodpecker_id(full: str, remote_id: str) -> str:
    code, looked = wp("GET", lookup_path(full))
    got = repo_id(looked)
    if got:
        return got
    code, body = wp("POST", activate_query(remote_id))
    got = repo_id(body)
    if got:
        return got
    if code != 409:
        raise RuntimeError(f"activate {full}: {code} {body}")
    code, looked = wp("GET", lookup_path(full))
    got = repo_id(looked)
    if not got:
        raise RuntimeError(f"lookup {full}: {code} {looked}")
    return got


def pin(wid: str) -> None:
    code, err = wp("PATCH", "/api/repos/" + wid, {"config_file": CONFIG})
    if code >= 400:
        raise RuntimeError(f"config {wid}: {code} {err}")


def main() -> None:
    if not ORG or not FJ_TOKEN or not WP_TOKEN:
        raise SystemExit("ACAHTI_ORG, ACAHTI_ADMIN_TOKEN, WOODPECKER_TOKEN required")
    names = org_repos()
    if not names:
        raise SystemExit(f"no repos in org {ORG}")
    ok = 0
    for full, remote_id in names:
        try:
            pin(woodpecker_id(full, remote_id))
        except RuntimeError as e:
            print(f"skip {full}: {e}", file=sys.stderr)
            continue
        ok += 1
        print(f"ok {full}", file=sys.stderr)
    if ok == 0:
        raise SystemExit("no repos activated")
    print(f"activated {ok}/{len(names)}", file=sys.stderr)


def selftest() -> None:
    raw = 'window.WOODPECKER_CSRF = "abc-123";\n'
    if parse_csrf(raw) != "abc-123":
        raise SystemExit("parse_csrf miss")
    if parse_csrf("no csrf here") != "":
        raise SystemExit("parse_csrf empty miss")
    if activate_query("42") != "/api/repos?forge_remote_id=42":
        raise SystemExit("activate_query")
    if activate_query("saidc/api-gateway") != "/api/repos?forge_remote_id=saidc%2Fapi-gateway":
        raise SystemExit("activate_query must not treat owner/name as a path")
    if lookup_path("saidc/api-gateway") != "/api/repos/lookup/saidc/api-gateway":
        raise SystemExit("lookup_path")
    if decode(b"") is not None:
        raise SystemExit("decode empty")
    if decode(b'{"id": 7}') != {"id": 7}:
        raise SystemExit("decode json")
    if decode(b"Repository is already active.") != "Repository is already active.":
        raise SystemExit("decode plain")
    if not isinstance(decode(b"{not-json"), str):
        raise SystemExit("decode broken json")
    if CONFIG != ".acahti/pipelines":
        raise SystemExit("CONFIG")
    print("ok", file=sys.stderr)


if __name__ == "__main__":
    if sys.argv[1:] == ["--selftest"]:
        selftest()
    else:
        main()
