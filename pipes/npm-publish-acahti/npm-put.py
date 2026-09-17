#!/usr/bin/env python3
"""PUT npm pack tarballs to an Acahti packages origin.

Env:
  ACAHTI_ADMIN_TOKEN or file ACAHTI_DATA/admin.token
  ACAHTI_ORG (default acme)
  ACAHTI_ADMIN_USER
  ORIGIN (default http://127.0.0.1:8080)
  HOST_HEADER / X_FORWARDED_PROTO for the public ROOT_URL
"""

import base64
import hashlib
import json
import os
import sys
import tarfile
import urllib.error
import urllib.request
from pathlib import Path


def package_json(tgz: Path) -> dict:
    with tarfile.open(tgz, "r:gz") as tf:
        for m in tf.getmembers():
            if m.name.endswith("package.json") and m.name.count("/") == 1:
                f = tf.extractfile(m)
                if f is None:
                    raise SystemExit(f"no package.json in {tgz}")
                return json.load(f)
    raise SystemExit(f"no package.json in {tgz}")


def main() -> None:
    if len(sys.argv) < 2:
        raise SystemExit("usage: npm-put.py <tgz> [...]")
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
        tgz = Path(raw)
        meta = package_json(tgz)
        name = meta["name"]
        version = meta["version"]
        blob = tgz.read_bytes()
        filename = tgz.name
        body = {
            "_id": name,
            "name": name,
            "description": meta.get("description", ""),
            "dist-tags": {"latest": version},
            "versions": {
                version: {
                    **meta,
                    "dist": {
                        "shasum": hashlib.sha1(blob).hexdigest(),
                        "integrity": "sha512-"
                        + base64.b64encode(hashlib.sha512(blob).digest()).decode(),
                        "tarball": filename,
                    },
                }
            },
            "_attachments": {
                filename: {
                    "content_type": "application/octet-stream",
                    "data": base64.b64encode(blob).decode(),
                    "length": len(blob),
                }
            },
        }
        encoded = urllib.request.quote(name, safe="@")
        url = f"{origin}/api/packages/{org}/npm/{encoded}"
        req = urllib.request.Request(
            url,
            data=json.dumps(body).encode(),
            method="PUT",
            headers={
                "Authorization": f"Basic {auth}",
                "Content-Type": "application/json",
                "Host": host,
                "User-Agent": "acahti-ops/1.0",
                "X-Forwarded-Proto": proto,
            },
        )
        try:
            with urllib.request.urlopen(req, timeout=120) as resp:
                print(name, version, resp.status)
        except urllib.error.HTTPError as e:
            err = e.read().decode("utf-8", "replace")[:500]
            if e.code == 409:
                print(name, version, 409, "already exists")
                continue
            print(name, version, e.code, err, file=sys.stderr)
            raise SystemExit(1)


if __name__ == "__main__":
    main()
