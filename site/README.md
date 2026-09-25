# Acahti product site

Static marketing site (no build). Multi-page layout per product packaging:

| Page | Path |
|---|---|
| Home | [`index.html`](./index.html) |
| Compare | [`compare.html`](./compare.html) |
| Security | [`security.html`](./security.html) |
| Self-host | [`self-host.html`](./self-host.html) |
| Cloud + waitlist | [`cloud.html`](./cloud.html) |
| Pricing | [`pricing.html`](./pricing.html) |

## Public host

After GitHub Pages is enabled for this repo (workflow [`.github/workflows/pages.yml`](../.github/workflows/pages.yml)):

**https://lpythu.github.io/acahti/**

Local preview:

```bash
python3 -m http.server 4173 --directory site
# http://127.0.0.1:4173/
```

Edit **Your instance** / **Org** on the home page — copy blocks refresh before clipboard write.
