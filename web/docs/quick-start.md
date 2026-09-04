# Quick start

This page gets you from zero to a published service. Commands assume a Linux/macOS shell; on Windows use the `.exe` the same way.

## 1. Install

One-line install (Linux / macOS):

```bash
curl -fsSL https://gtunnel.dev/install.sh | sh
```

The script detects your OS and CPU architecture, downloads the matching binary from [GitHub Releases](https://github.com/zxln007/gt/releases) and installs it to `/usr/local/bin` (or `~/.local/bin` without root). Verify with `gt --version`.

Alternatives:

- **Direct download** — static binaries for Windows / macOS / Linux (x86_64, arm64, RISC-V) on the [download page](/).
- **Desktop app** — GT-Desktop wraps the same engine in a GUI if you prefer clicks over commands.
- **Docker** — `ghcr.io/ao-space/gt:server-dev` and `gt:client-dev` images exist for server deployment; see [Docker deployment](examples.md#self-host-the-relay-in-docker).

## 2. The five-minute tour (zero config)

Both roles start with no arguments and open a local web console for editing their config:

```bash
gt server
```

The log prints the console address — `http://127.0.0.1:8000` by default. Open it, and you can configure listeners, users and services in a form instead of YAML. The same works for the client on port `7000`:

```bash
gt client
```

!> Config edited in the web console is saved to the default config path, so it survives restarts. This is the recommended way to build your config; the YAML below is what it produces.

## 3. Publish your first service

Prefer the terminal? Two terminals, three commands.

**Terminal 1 — start a relay server** (`-id`/`-secret` register a client credential):

```bash
gt server -addr 12080 -id id1 -secret secret1
```

**Terminal 2 — connect a client** with a local service to publish:

```bash
gt client -local http://127.0.0.1:8080 -remote tcp://127.0.0.1:12080 -id id1 -secret secret1
```

**Test it.** The server routes HTTP by host prefix — the client's `id` is the prefix — so ask curl to send a matching `Host` header:

```bash
curl -H "Host: id1.example.com" http://127.0.0.1:12080/
```

You should get the response of your local service on `127.0.0.1:8080`. On a real server with a domain, visitors would simply open `http://id1.your-domain:12080/`.

## 4. Where to go from here

- Real-world scenarios (SSH behind NAT, several services at once, P2P): [Usage examples](examples.md)
- Every flag and YAML option: [Configuration](config.md)
- Something not working: [FAQ](faq.md)
