#!/usr/bin/env python3
"""Activate every org repo in Woodpecker and pin .acahti/pipelines."""

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


def parse_csrf(raw: str) -> str:
    i = raw.find(CSRF_MARK)
    if i < 0:
        return ""
    s = raw[i + len(CSRF_MARK) :]
    j = s.find('"')
    if j < 0:
        return ""
    return s[:j]


def csrf() -> str:
    req = urllib.request.Request(
        WP + "/ci/web-config.js",
        headers={"Cookie": "user_sess=" + WP_TOKEN},
    )
    try:
        with urllib.request.urlopen(req) as resp:
            return parse_csrf(resp.read().decode("utf-8", "replace"))
    except urllib.error.HTTPError:
        return ""


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
        "Content-Type": "application/json",
    }
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
            raw = resp.read()
            return resp.status, json.loads(raw) if raw else None
    except urllib.error.HTTPError as e:
        raw = e.read()
        parsed: object
        try:
            parsed = json.loads(raw) if raw else e.reason
        except json.JSONDecodeError:
            parsed = raw.decode("utf-8", "replace")[:300]
        return e.code, parsed


def org_repos() -> list[str]:
    page = 1
    names: list[str] = []
    while True:
        rows = fj(f"/api/v1/orgs/{urllib.parse.quote(ORG)}/repos?page={page}&limit=50")
        if not isinstance(rows, list) or not rows:
            break
        for row in rows:
            if isinstance(row, dict) and row.get("full_name"):
                names.append(str(row["full_name"]))
        if len(rows) < 50:
            break
        page += 1
    return names


def main() -> None:
    if not ORG or not FJ_TOKEN or not WP_TOKEN:
        raise SystemExit("ACAHTI_ORG, ACAHTI_ADMIN_TOKEN, WOODPECKER_TOKEN required")
    names = org_repos()
    if not names:
        raise SystemExit(f"no repos in org {ORG}")
    ok = 0
    for full in names:
        code, body = wp("POST", "/api/repos?forge_remote_id=" + urllib.parse.quote(full))
        if code >= 400:
            code, body = wp("POST", "/api/repos/" + urllib.parse.quote(full))
        if isinstance(body, dict) and body.get("id"):
            rid = str(body["id"])
        else:
            code, listed = wp("GET", "/api/user/repos?page=1&perPage=50")
            rid = ""
            if isinstance(listed, list):
                for repo in listed:
                    if isinstance(repo, dict) and repo.get("full_name") == full:
                        rid = str(repo.get("id") or "")
                        break
            if not rid:
                print(f"skip {full}: activate {code} {body}", file=sys.stderr)
                continue
        patch, err = wp("PATCH", "/api/repos/" + rid, {"config": CONFIG, "active": True})
        if patch >= 400:
            print(f"skip {full}: config {patch} {err}", file=sys.stderr)
            continue
        ok += 1
        print(f"ok {full}", file=sys.stderr)
    if ok == 0:
        raise SystemExit("no repos activated")
    print(f"activated {ok}/{len(names)}", file=sys.stderr)


def selftest() -> None:
    raw = 'window.WOODPECKER_CSRF = "abc-123";\n'
    got = parse_csrf(raw)
    if got != "abc-123":
        raise SystemExit(f"parse_csrf -> {got!r} want 'abc-123'")
    if parse_csrf("no csrf here") != "":
        raise SystemExit("parse_csrf empty miss")
    print("ok", file=sys.stderr)


if __name__ == "__main__":
    if sys.argv[1:] == ["--selftest"]:
        selftest()
    else:
        main()
