#!/usr/bin/env python3
"""Ensure the Forgejo OAuth app Woodpecker uses (loopback redirect only).

Reuse the existing acahti-ci client when .env still matches Forgejo.
Only rotate when the app is missing or the stored client id is stale.
Prints JSON {client_id, client_secret}.
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
    have_id = os.environ.get("WOODPECKER_FORGEJO_CLIENT", "").strip()
    have_secret = os.environ.get("WOODPECKER_FORGEJO_SECRET", "").strip()
    apps = req("GET", "/api/v1/user/applications/oauth2") or []
    if not isinstance(apps, list):
        raise SystemExit("oauth app list is not an array")
    live = [a for a in apps if isinstance(a, dict) and a.get("name") == NAME]
    if have_id and have_secret:
        for app in live:
            if app.get("client_id") == have_id:
                json.dump({"client_id": have_id, "client_secret": have_secret}, sys.stdout)
                sys.stdout.write("\n")
                return
    for app in live:
        req("DELETE", f"/api/v1/user/applications/oauth2/{app['id']}")
    created = req("POST", "/api/v1/user/applications/oauth2", {
        "name": NAME,
        "confidential_client": True,
        "redirect_uris": [REDIRECT],
    })
    if not isinstance(created, dict) or not created.get("client_id") or not created.get("client_secret"):
        raise SystemExit("create oauth app returned no client")
    json.dump({"client_id": created["client_id"], "client_secret": created["client_secret"]}, sys.stdout)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
