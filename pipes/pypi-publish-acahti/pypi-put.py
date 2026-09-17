#!/usr/bin/env python3
"""PUT PyPI artifacts to an Acahti packages origin.

Env:
  ACAHTI_ADMIN_TOKEN or file ACAHTI_DATA/admin.token
  ACAHTI_ORG (default acme)
  ACAHTI_ADMIN_USER
  ORIGIN (default http://127.0.0.1:8080)
  HOST_HEADER / X_FORWARDED_PROTO for the public ROOT_URL
"""

import base64
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


def main() -> None:
    if len(sys.argv) < 2:
        raise SystemExit("usage: pypi-put.py <wheel-or-sdist> [...]")
    data_root = Path(os.environ.get("ACAHTI_DATA", "/var/lib/acahti"))
    token = os.environ.get("ACAHTI_ADMIN_TOKEN", "").strip()
    if not token:
        token = (data_root / "admin.token").read_text().strip()
    user = os.environ.get("ACAHTI_ADMIN_USER", "acahti")
    org = os.environ.get("ACAHTI_ORG", "acme")
    origin = os.environ.get("ORIGIN", "http://127.0.0.1:8080").rstrip("/")
    host = os.environ.get("HOST_HEADER", os.environ.get("DOMAIN", "localhost"))
    proto = os.environ.get("X_FORWARDED_PROTO", "https")
    auth = base64.b64encode(f"{user}:{token}".encode()).decode()
    for raw in sys.argv[1:]:
        art = Path(raw)
        blob = art.read_bytes()
        filename = art.name
        url = (
            f"{origin}/api/packages/{org}/pypi"
            f"?filename={urllib.parse.quote(filename)}"
        )
        req = urllib.request.Request(
            url,
            data=blob,
            method="PUT",
            headers={
                "Authorization": f"Basic {auth}",
                "Content-Type": "application/octet-stream",
                "Host": host,
                "User-Agent": "acahti-ops/1.0",
                "X-Forwarded-Proto": proto,
            },
        )
        try:
            with urllib.request.urlopen(req, timeout=120) as resp:
                print(filename, resp.status)
        except urllib.error.HTTPError as e:
            err = e.read().decode("utf-8", "replace")[:500]
            print(filename, e.code, err, file=sys.stderr)
            raise SystemExit(1)


if __name__ == "__main__":
    main()
