#!/usr/bin/env python3
"""Ensure the Forgejo OAuth app Woodpecker uses (loopback redirect only).

Prints JSON {client_id, client_secret}. Secret is reused from the
environment when the app already exists; otherwise a new app is created.
"""

import json
import os
import sys
import urllib.error
import urllib.request

NAME = "acahti-ci"
REDIRECT = "http://127.0.0.1:8000/ci/authorize"
FJ = os.environ.get("FORGEJO_LOOPBACK", "http://127.0.0.1:3000").rstrip("/")
TOKEN = os.environ.get("ACAHTI_ADMIN_TOKEN", "").strip()


def req(method: str, path: str, data: dict | None = None) -> object:
    body = None if data is None else json.dumps(data).encode()
    request = urllib.request.Request(
        FJ + path,
        data=body,
        headers={"Authorization": "token " + TOKEN, "Content-Type": "application/json"},
        method=method,
    )
    try:
        with urllib.request.urlopen(request) as resp:
            raw = resp.read()
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as e:
        raise SystemExit(f"HTTP {e.code} {method} {path}: {e.read().decode()[:300]}") from e


def main() -> None:
    if not TOKEN:
        raise SystemExit("ACAHTI_ADMIN_TOKEN is empty")
    payload = {"name": NAME, "confidential_client": True, "redirect_uris": [REDIRECT]}
    apps = req("GET", "/api/v1/user/applications/oauth2") or []
    if not isinstance(apps, list):
        raise SystemExit("oauth app list is not an array")
    mine = [a for a in apps if isinstance(a, dict) and a.get("name") == NAME]
    have = os.environ.get("WOODPECKER_FORGEJO_CLIENT", "").strip()
    secret = os.environ.get("WOODPECKER_FORGEJO_SECRET", "").strip()
    if have and secret and mine:
        for app in mine:
            req("PATCH", f"/api/v1/user/applications/oauth2/{app['id']}", payload)
        json.dump({"client_id": have, "client_secret": secret}, sys.stdout)
        sys.stdout.write("\n")
        return
    for app in mine:
        req("DELETE", f"/api/v1/user/applications/oauth2/{app['id']}")
    created = req("POST", "/api/v1/user/applications/oauth2", payload)
    if not isinstance(created, dict) or not created.get("client_id") or not created.get("client_secret"):
        raise SystemExit("create oauth app returned no client")
    json.dump({"client_id": created["client_id"], "client_secret": created["client_secret"]}, sys.stdout)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
