# G-Tunnel Documentation

G-Tunnel (`gt`) is a high-performance relay proxy: it carries your traffic over WebSocket(s)/HTTP(s)/TCP between a public relay server and clients running next to private services. A Rust async data plane paired with a Go control plane keeps it fast under load — wrk-tested at up to 6x frp throughput with zero errors under full concurrency.

Everything in these docs applies to the same single `gt` binary; `gt server` and `gt client` just start different roles.

## How it works

Visitors reach the public relay, which recognizes the target by hostname prefix or TCP port and hands the traffic to the client that published it. The client forwards to the actual local service:

```text
      ┌──────────────────────────────────────┐
      │  Web    Android     iOS    PC    ... │
      └──────────────────┬───────────────────┘
                  ┌──────┴──────┐
                  │  GT Server  │   public relay
                  └──────┬──────┘
       ┌─────────────────┼─────────────────┐
┌──────┴──────┐   ┌──────┴──────┐   ┌──────┴──────┐
│  GT Client  │   │  GT Client  │   │  GT Client  │ ...
└──────┬──────┘   └──────┬──────┘   └──────┬──────┘
┌──────┴──────┐   ┌──────┴──────┐   ┌──────┴──────┐
│     SSH     │   │   HTTP(S)   │   │     SMB     │ ...
└─────────────┘   └─────────────┘   └─────────────┘
```

Two addressing schemes decide who gets the traffic:

- **Host prefix** — for HTTP(S) services. The server reads the `Host` header (e.g. `myapp.relay.example.com`) and routes by the first `.`-separated prefix. By default a client's `id` is its prefix, so `-id myapp` publishes `myapp.<relay-domain>`.
- **TCP port** — for anything that speaks TCP (SSH, SMB, databases, game servers). The server opens a port and pipes it through to the client's local service.

In P2P mode the relay only brokers the connection; data then flows directly between the two clients over WebRTC. See [P2P direct connect](examples.md#p2p-direct-connect).

## Key concepts

| Concept | What it is |
|---------|------------|
| GT Server | The relay. Needs a machine with a public IP (VPS, cloud host) — or use the hosted relay at [console.gtunnel.dev](https://console.gtunnel.dev) |
| GT Client | Runs next to the private service and dials out to the relay. Works behind NAT with no port forwarding |
| id / secret | Client credentials. The id doubles as the host prefix |
| Web console | Start `gt server` or `gt client` with no arguments and edit the config in a browser instead of YAML |
| GT-Desktop | Native GUI wrapping the same engine — see the [homepage](/) |

## Where to go next

- [Quick start](quick-start.md) — install and publish your first service in five minutes
- [Usage examples](examples.md) — web publishing, SSH, multi-service configs, QUIC/TLS, P2P, Docker deployment
- [Configuration](config.md) — every option for client and server, plus signals and batch startup
- [FAQ](faq.md) — troubleshooting and design notes
