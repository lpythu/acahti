# Acahti product site

Static marketing homepage (no build step). Open `index.html` locally or publish this folder to GitHub Pages / Cloudflare Pages.

## Why static HTML

- Humans get a visual, animated landing with one-click copy.
- Agents can `curl` / read the HTML and the linked `docs/product/*.md` without a JS framework.
- Instance UX (real URLs) lives in the app home page (`web` → `/`); this site is the public pitch.

## Local preview

```bash
# from repo root
python3 -m http.server 4173 --directory site
# open http://127.0.0.1:4173/
```

Asset paths are self-contained under `site/assets/`.

## Edit host/org

On the page, change **Your instance** and **Org** — copy blocks update before clipboard write.
