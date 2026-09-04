# FAQ

## What hostname do visitors use?

`<prefix>.<relay-domain>` — e.g. `myapp.relay.example.com`. The prefix defaults to your client `id`; override or add more with `-hostPrefix` / `services[].hostPrefix`. On shared relays the operator may enable `host.withID`, which namespaces prefixes as `id-host` (`myapp-web.relay.example.com`) to avoid collisions between users.

## I started `gt server` with no arguments — now what?

Open the printed web console address (`http://127.0.0.1:8000` by default; clients use `http://127.0.0.1:7000`). Configure everything there; the config is saved and reused on the next start.

## Can I change config without dropping connections?

Yes — edit the config, then run `gt -s reload`. Work processes re-read the config while forwarding continues. Full restarts (`gt -s restart`) and binary upgrades also keep forwarding alive; soak tests measured 0 errors under load during reload.

## `-remoteTCPRandom` picked which port?

The server assigns one from the user's configured pool (`users[].tcp[].range`, e.g. `30000-39999`, bounded by `tcpNumber`). The client log prints the mapped address at startup — run with `-logLevel debug` if you don't see it.

## P2P direct connect doesn't establish

In order of likelihood:

1. **Symmetric NAT on either side** — expected failure; there is no TURN fallback for P2P. Traffic falls back to the relay.
2. **UDP blocked** — hole punching needs UDP; allow the `gt` process in both firewalls (Windows Defender included).
3. **Version mismatch** — both ends must run the same build.
4. Debug with `-logLevel debug` on both clients: `typ srflx` candidates on both sides mean STUN worked; if the pair selects `relay`, you are seeing the fallback.

## How do I protect my relay?

- Only register credentials you handed out yourself (never `allowAnyClient` on a public address)
- Put the web console on `127.0.0.1` or behind auth (`admin`, `password`, `signingKey`)
- Give each user quotas: `speed`, `connections`, `tcpNumber`, `host.number`
- Prefer `tls://`/`quic://` listeners so credentials never cross the wire in clear text

Secrets and traffic content are never written to logs by design.

## Which platforms are supported?

Windows (x86_64), macOS (Apple Silicon, Intel), Linux (x86_64, arm64, RISC-V) as static single binaries from [GitHub Releases](https://github.com/zxln007/gt/releases); Docker images for server and client. The same engine ships inside [GT-Desktop](/) for Windows and macOS.

## How does it compare to frp?

Both do reverse tunneling. G-Tunnel's wrk benchmarks show up to 6x throughput and roughly 1/8 the average latency with zero errors under full load, plus built-in QUIC and P2P — see [Benchmarks](/#benchmark) for methodology and raw output.
