# G-Tunnel Portal Deployment Guide

This directory contains the static website homepage files and a `Dockerfile` for serving the G-Tunnel Portal.

## Site Structure

- `index.html` — English homepage (site root, default for overseas visitors)
- `zh/index.html` — Chinese homepage
- `assets/` — shared stylesheet, script and icons (zero external CDN dependencies)
- `docs/` — English documentation, powered by docsify: **`index.html` is only a shell; the content lives in the Markdown files next to it**. To add or edit docs, just write Markdown (`quick-start.md`, `examples.md`, …) and list new pages in `_sidebar.md` — no HTML, no build step
- `zh/docs/` — Chinese documentation, same docsify setup (shell + Markdown). It shares `docs/theme.css` and the vendored scripts under `docs/vendor/`
- `docs/vendor/` — self-hosted docsify runtime (docsify, search plugin, copy-code plugin, Prism grammars; MIT). Vendored so the site keeps its zero-external-CDN guarantee. Note the script order in the shells: docsify bundles the Prism core and exposes `window.Prism`, so grammar files must load *after* `docsify.min.js`
- `nginx.conf` — nginx site config: serves `*.md` as `text/markdown` with `Cache-Control: no-cache` so doc edits show up immediately
- `install.sh` — one-line installer served at `/install.sh`:
  `curl -fsSL https://gtunnel.dev/install.sh | sh`
  Downloads the matching binary from GitHub Releases (`zxln007/gt` by default; override with `GT_REPO` / `GT_VERSION` / `GT_BIN_DIR` env vars) and verifies sha256 when the release ships `checksums.txt`.
- Download buttons link directly to GitHub Release assets; no local `/downloads/` volume needed.

## Quick Start (Docker)

To build and run this portal inside a Docker container:

### 1. Build the Docker Image
Navigate to this `web` directory and build the image:
```bash
docker build -t g-tunnel-client-portal .
```

### 2. Run the Container
To run the container and map it to a local port (e.g., `8080`):
```bash
docker run -d -p 8080:80 --name g-tunnel-portal g-tunnel-client-portal
```

### 3. Open in Browser
Open your browser and navigate to:
[http://localhost:8080](http://localhost:8080)

---

## Local Development (Without Docker)

Since the website is built using modern **Vanilla HTML5 & CSS3** with zero compile steps, you can directly open `index.html` in any web browser to view the page.

To run it locally with a development server (e.g., using python's built-in server):
```bash
# Python 3, serve from the web/ directory so /assets/ and /zh/ resolve
python -m http.server 8000
```
Then visit `http://localhost:8000` (English) or `http://localhost:8000/zh/` (Chinese).
