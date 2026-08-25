# G-Tunnel Portal Deployment Guide

This directory contains the static website homepage files and a `Dockerfile` for serving the G-Tunnel Portal.

## Quick Start (Docker)

To build and run this portal inside a Docker container:

### 1. Build the Docker Image
Navigate to this `web` directory and build the image:
```bash
docker build -t g-tunnel-client-portal .
```

### 2. Run the Container
To run the container and map it to a local port (e.g., `8080`), you can mount your local directory containing the compiled G-Tunnel client binaries (such as `.zip`, `.dmg`, `.tar.gz` files) directly into the Nginx serving root:
```bash
docker run -d \
  -p 8080:80 \
  -v /path/to/local/binaries:/usr/share/nginx/html/downloads \
  --name g-tunnel-portal \
  g-tunnel-client-portal
```

### 3. Open in Browser
Open your browser and navigate to:
[http://localhost:8080](http://localhost:8080)

---

## Local Development (Without Docker)

Since the website is built using modern **Vanilla HTML5 & CSS3** with zero compile steps, you can directly open `index.html` in any web browser to view the page.

To run it locally with a development server (e.g., using python's built-in server):
```bash
# Python 3
python -m http.server 8000
```
Then visit `http://localhost:8000`.
