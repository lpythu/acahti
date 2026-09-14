#!/usr/bin/env python3
"""Complete Woodpecker → Forgejo OAuth on the acahti host (no browser).

Talks to loopback ports published by compose. Rewrites compose DNS
(forgejo / woodpecker) and public ROOT_URL to 127.0.0.1. Prints a
Woodpecker personal token on stdout.
"""

import http.cookiejar
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request


def env(name: str, default: str = "") -> str:
    return os.environ.get(name, default)


ROOT = env("ROOT_URL", "https://acahti.example.com").rstrip("/")
DOMAIN = env("DOMAIN", "localhost")
USER = env("ACAHTI_ADMIN_USER", "acahti")
PASSWORD = env("ACAHTI_ADMIN_PASSWORD", "")
FJ = env("FORGEJO_LOOPBACK", "http://127.0.0.1:3000").rstrip("/")
WP = env("WOODPECKER_LOOPBACK", "http://localhost:8000").rstrip("/")


class LoopbackPolicy(http.cookiejar.DefaultCookiePolicy):
    def return_ok_secure(self, cookie, request):
        return True

    def set_ok_secure(self, cookie, request):
        return True

    def return_ok_domain(self, cookie, request):
        return True

    def set_ok_domain(self, cookie, request):
        return True

    def return_ok_port(self, cookie, request):
        return True

    def set_ok_port(self, cookie, request):
        return True


class RewriteRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return super().redirect_request(req, fp, code, msg, headers, rewrite(newurl))


def opener() -> urllib.request.OpenerDirector:
    cj = http.cookiejar.CookieJar(policy=LoopbackPolicy())
    return urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(cj),
        RewriteRedirect(),
    )


def rewrite(url: str) -> str:
    if not url:
        return url
    u = url.replace("http://forgejo:3000", FJ).replace("http://woodpecker:8000", WP)
    host = (urllib.parse.urlparse(u).hostname or "").lower()
    if host in {"127.0.0.1", "localhost"}:
        return u
    parsed = urllib.parse.urlparse(u)
    path = parsed.path or ""
    if path.startswith("/ci") or path.startswith("/authorize"):
        suffix = path if path.startswith("/ci") else "/ci" + path
        if parsed.query:
            suffix += "?" + parsed.query
        return WP + suffix
    if path.startswith("/login/oauth") or path.startswith("/user/login"):
        suffix = path
        if parsed.query:
            suffix += "?" + parsed.query
        return FJ + suffix
    return u


def read(op: urllib.request.OpenerDirector, url: str, data: bytes | None = None, headers: dict | None = None) -> tuple[str, str]:
    url = rewrite(url)
    req = urllib.request.Request(url, data=data, method="POST" if data else "GET")
    req.add_header("User-Agent", "acahti-configure")
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    try:
        with op.open(req, timeout=30) as resp:
            final = resp.geturl()
            body = resp.read().decode("utf-8", "replace")
            return final, body
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", "replace")
        if e.code in (301, 302, 303, 307, 308):
            loc = e.headers.get("Location", "")
            return read(op, rewrite(loc), headers=headers)
        raise SystemExit(f"HTTP {e.code} {url}: {body[:400]}")


def csrf(html: str) -> str:
    m = re.search(r'name="_csrf"\s+value="([^"]+)"', html)
    if not m:
        m = re.search(r'csrfToken["\s:=]+["\']([^"\']+)', html)
    if not m:
        raise SystemExit("no csrf on login page")
    return m.group(1)


def main() -> None:
    if not PASSWORD:
        raise SystemExit("ACAHTI_ADMIN_PASSWORD is empty")
    op = opener()
    _, login_html = read(op, f"{FJ}/user/login")
    form = {"user_name": USER, "password": PASSWORD}
    if "_csrf" in login_html:
        form["_csrf"] = csrf(login_html)
    payload = urllib.parse.urlencode(form).encode()
    read(op, f"{FJ}/user/login", data=payload, headers={"Content-Type": "application/x-www-form-urlencoded"})

    final, body = read(op, f"{WP}/ci/authorize")
    if 'id="authorize-app"' in body or 'name="granted"' in body:
        form = {"granted": "true"}
        if "_csrf" in body:
            form["_csrf"] = csrf(body)
        for name in ("client_id", "redirect_uri", "response_type", "state", "scope", "nonce"):
            m = re.search(rf'name="{name}"\s+value="([^"]*)"', body)
            if m:
                form[name] = m.group(1)
        grant_url = f"{FJ}/login/oauth/grant"
        mact = re.search(r'<form[^>]+action="([^"]*oauth[^"]*)"', body)
        if mact:
            grant_url = mact.group(1)
            if grant_url.startswith("/"):
                grant_url = FJ + grant_url
            grant_url = rewrite(grant_url)
        final, body = read(
            op,
            grant_url,
            data=urllib.parse.urlencode(form).encode(),
            headers={"Content-Type": "application/x-www-form-urlencoded"},
        )

    read(op, f"{WP}/ci/")
    names: list[str] = []
    for handler in op.handlers:
        jar = getattr(handler, "cookiejar", None)
        if not jar:
            continue
        for cookie in jar:
            names.append(f"{cookie.name}@{cookie.domain}")
            if cookie.name == "user_sess" and cookie.value:
                sys.stdout.write(cookie.value.strip() + "\n")
                return
    raise SystemExit(f"woodpecker user_sess cookie missing after OAuth; cookies={names}")


def selftest() -> None:
    cases = {
        "http://forgejo:3000/login/oauth/authorize": f"{FJ}/login/oauth/authorize",
        "http://woodpecker:8000/ci/authorize": f"{WP}/ci/authorize",
        "http://127.0.0.1:8000/ci/authorize": "http://127.0.0.1:8000/ci/authorize",
        "http://localhost:8000/ci/authorize": "http://localhost:8000/ci/authorize",
        f"{ROOT}/login/oauth/authorize": f"{FJ}/login/oauth/authorize",
        f"{ROOT}/ci/authorize": f"{WP}/ci/authorize",
    }
    for raw, want in cases.items():
        got = rewrite(raw)
        if got != want:
            raise SystemExit(f"rewrite {raw!r} -> {got!r} want {want!r}")
    print("ok", file=sys.stderr)


if __name__ == "__main__":
    if sys.argv[1:] == ["--selftest"]:
        selftest()
    else:
        main()
