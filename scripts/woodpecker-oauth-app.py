#!/usr/bin/env python3
"""Ensure the Forgejo OAuth app Woodpecker uses (loopback redirect only).

Always deletes acahti-ci and creates it again so the secret in
.env matches Forgejo. Prints JSON {client_id, client_secret}.
"""

import json
import os
import sys
import urllib.error
import urllib.request

NAME = "acahti-ci"
REDIRECT = "http://localhost:8000/ci/authorize"
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
    for app in apps:
        if isinstance(app, dict) and app.get("name") == NAME:
            req("DELETE", f"/api/v1/user/applications/oauth2/{app['id']}")
    created = req("POST", "/api/v1/user/applications/oauth2", payload)
    if not isinstance(created, dict) or not created.get("client_id") or not created.get("client_secret"):
        raise SystemExit("create oauth app returned no client")
    json.dump({"client_id": created["client_id"], "client_secret": created["client_secret"]}, sys.stdout)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
