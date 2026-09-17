#!/usr/bin/env python3
"""POST PyPI artifacts to an Acahti packages origin (Gitea/twine upload).

Env:
  ACAHTI_ADMIN_TOKEN or file ACAHTI_DATA/admin.token
  ACAHTI_ORG (default acme)
  ACAHTI_ADMIN_USER
  ORIGIN (default http://127.0.0.1:8080)
  HOST_HEADER / X_FORWARDED_PROTO for the public ROOT_URL
"""

import base64
import hashlib
import os
import sys
import urllib.error
import urllib.request
import uuid
from pathlib import Path


def pep_name_version(filename: str) -> tuple[str, str]:
    if filename.endswith(".whl"):
        stem = filename[: -len(".whl")]
        parts = stem.split("-")
        if len(parts) < 5:
            raise SystemExit(f"unrecognized wheel name: {filename}")
        return parts[0], parts[1]
    if filename.endswith(".tar.gz"):
        stem = filename[: -len(".tar.gz")]
        name, sep, version = stem.rpartition("-")
        if not sep:
            raise SystemExit(f"unrecognized sdist name: {filename}")
        return name, version
    raise SystemExit(f"unsupported artifact: {filename}")


def multipart(fields: dict[str, str], filename: str, blob: bytes) -> tuple[bytes, str]:
    boundary = uuid.uuid4().hex
    body = bytearray()
    for key, value in fields.items():
        body.extend(f"--{boundary}\r\n".encode())
        body.extend(f'Content-Disposition: form-data; name="{key}"\r\n\r\n'.encode())
        body.extend(value.encode())
        body.extend(b"\r\n")
    body.extend(f"--{boundary}\r\n".encode())
    body.extend(
        f'Content-Disposition: form-data; name="content"; filename="{filename}"\r\n'.encode()
    )
    body.extend(b"Content-Type: application/octet-stream\r\n\r\n")
    body.extend(blob)
    body.extend(b"\r\n")
    body.extend(f"--{boundary}--\r\n".encode())
    return bytes(body), boundary


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
    url = f"{origin}/api/packages/{org}/pypi"
    for raw in sys.argv[1:]:
        art = Path(raw)
        blob = art.read_bytes()
        filename = art.name
        name, version = pep_name_version(filename)
        digest = hashlib.sha256(blob).hexdigest()
        body, boundary = multipart(
            {
                "name": name,
                "version": version,
                "sha256_digest": digest,
            },
            filename,
            blob,
        )
        req = urllib.request.Request(
            url,
            data=body,
            method="POST",
            headers={
                "Authorization": f"Basic {auth}",
                "Content-Type": f"multipart/form-data; boundary={boundary}",
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
