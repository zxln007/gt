# Usage examples

Ready-made recipes for common setups. All examples assume:

- a relay reachable at `relay.example.com` — your own server, or the hosted relay whose exact address you get from [console.gtunnel.dev](https://console.gtunnel.dev)
- client credentials obtained from the relay operator (for the hosted relay: the console issues them with one click)

Substitute your own values everywhere. See [Configuration](config.md) for the full option list.

## Publish a local web app under your relay domain

The classic reverse-tunnel: a web service on a machine behind NAT becomes reachable at a URL of the relay.

```bash
gt client -local http://127.0.0.1:3000 \
  -remote tcp://relay.example.com:7000 \
  -id myapp -secret s3cret
```

Visitors open **http://myapp.relay.example.com:7000** — the client's `id` (`myapp`) is the hostname prefix the server routes by. Point `*.relay.example.com` at the relay's IP and any number of clients can publish under it.

!> Only the first packet's `Host` header is inspected to pick the target; after that the data stream is forwarded as-is. That is a deliberate privacy/performance design of the relay.

Need a different prefix than the `id`? Add `-hostPrefix staging`, or publish several names from one client — see [several services from one client](#several-services-from-one-client).

## SSH into a machine behind NAT

Publish a local SSH daemon on a relay TCP port and `ssh` to it from anywhere:

```bash
gt client -local tcp://127.0.0.1:22 \
  -remote tcp://relay.example.com:7000 \
  -id home -secret s3cret \
  -remoteTCPPort 6022
```

From any machine:

```bash
ssh -p 6022 user@relay.example.com
```

Works for any TCP service — replace `tcp://127.0.0.1:22` with e.g. `tcp://127.0.0.1:445` (SMB), `tcp://127.0.0.1:5432` (PostgreSQL) or `tcp://127.0.0.1:25565` (Minecraft).

**Random port variant.** If you can't fix a port in advance, let the server assign one from its pool: use `-remoteTCPRandom` instead of `-remoteTCPPort`. The server must be configured with a port range (e.g. `-tcpRange 30000-39999`); the client log prints the assigned port at startup.

## Several services from one client

One client process can publish many services via a config file instead of flags:

```yaml
# client.yaml
id: home
secret: s3cret
remote: tcp://relay.example.com:7000

services:
  - hostPrefix: web          # -> http://web.relay.example.com:7000
    local: http://127.0.0.1:8080
  - hostPrefix: nas          # -> http://nas.relay.example.com:7000
    local: http://127.0.0.1:5000
  - local: tcp://127.0.0.1:22   # -> a random relay TCP port (see log)
    remoteTCPRandom: true
```

```bash
gt client -c ./client.yaml
```

Each `services` entry can set `hostPrefix`, `local`, `remoteTCPPort`, `remoteTCPRandom`, `localTimeout` and `useLocalAsHTTPHost` (send the local address as the HTTP `Host` header — useful when the backend serves multiple virtual hosts).

?> If the server runs with `host.withID: true` (common on shared relays), prefixes are namespaced as `id-host` — e.g. `home-web.relay.example.com`. Check with your relay operator.

## QUIC between client and relay

On lossy or high-latency uplinks (mobile networks, long-distance VPS links), run the client↔relay leg over QUIC. The server needs a TLS certificate for that listener:

```bash
gt server -addr 7000 -quicAddr 443 \
  -certFile /etc/gt/tls.crt -keyFile /etc/gt/tls.key \
  -id id1 -secret secret1
```

```bash
gt client -local http://127.0.0.1:8080 \
  -remote quic://relay.example.com:443 \
  -remoteCertInsecure \
  -id id1 -secret secret1
```

`-remoteCertInsecure` accepts the server's self-signed certificate; for production, drop it and pin the certificate instead with `-remoteCert /path/to/tls.crt`.

In wrk benchmarks the QUIC transport kept 128k req/s and brought average latency on short requests down to 1.84ms — see [Benchmarks](/#benchmark) on the homepage. Enable BBR congestion control on both ends with `-bbr` if the link is very lossy.

## TLS between client and relay

Same idea over stream TLS — useful where UDP is blocked:

```bash
gt server -addr 7000 -tlsAddr 443 \
  -certFile /etc/gt/tls.crt -keyFile /etc/gt/tls.key \
  -id id1 -secret secret1
```

```bash
gt client -local http://127.0.0.1:8080 \
  -remote tls://relay.example.com:443 \
  -id id1 -secret secret1
```

?> This TLS leg protects the tunnel itself. If you also want end-to-end TLS for *visitors* under your own domain, terminate HTTPS on the server (`-sniAddr` forwards raw TLS by SNI without touching the payload), or put your usual reverse proxy (Caddy/nginx) in front of the published URL.

## P2P direct connect

Two clients behind different NATs can exchange traffic directly over WebRTC — the relay only brokers the handshake (and falls back to relaying if a hole can't be punched). Example topology: a file server behind NAT-A ("provider"), a gateway behind NAT-B exposing it on a local port.

**Relay server** (public VPS, also providing STUN):

```bash
gt server -addr 0.0.0.0:7000 -stunAddr 0.0.0.0:3478 \
  -id prov -secret prov-secret
```

**Provider** (holds the real service, behind NAT-A):

```bash
python3 -m http.server 8080   # any local service

gt client -id prov -secret prov-secret \
  -remote <server_ip>:7000 \
  -remoteSTUN stun:<server_ip>:3478 \
  -local http://127.0.0.1:8080 \
  -webrtcThreadMode
```

**Gateway** (behind NAT-B, exposes the provider locally):

```bash
gt client -id gw -secret gw-secret \
  -remote <server_ip>:7000 \
  -remoteSTUN stun:<server_ip>:3478 \
  -tcpForwardAddr 0.0.0.0:8888 \
  -tcpForwardHostPrefix prov
```

Anyone on the gateway's LAN opens `http://<gateway-ip>:8888/` and gets the provider's content. Check `-logLevel debug` on both clients: seeing `typ srflx` candidates on both sides means the direct P2P path was established.

NAT compatibility at a glance:

| NAT combination | Result |
|---|---|
| Full-cone × full-cone (most home links) | Direct connect succeeds |
| Restricted-cone × restricted-cone | Succeeds |
| Either side behind a symmetric NAT | Fails — no TURN relay for P2P, falls back to relayed traffic |
| Either side has a public IP | Succeeds |

!> Allow inbound/outbound **UDP** for the `gt` process in your firewall — hole punching needs it.

## Self-host the relay in Docker

The `server-dev` image generates a config from environment variables at startup:

```bash
docker run -d --name gt-server --restart unless-stopped \
  -p 80:80 -p 443:443 -p 3478:3478/udp \
  -v /opt/gt/crt:/opt/crt:ro \
  -e NETWORK_ADDR=80 \
  -e NETWORK_TLSADDR=443 \
  -e NETWORK_STUNADDR=3478 \
  -e NETWORK_WEB_ADDR=0.0.0.0:8000 \
  ghcr.io/ao-space/gt:server-dev
```

- `-v .../crt` mounts your certificate as `/opt/crt/tls.crt` + `tls.key` (used by the TLS and QUIC listeners)
- open `http://<vps>:8000` for the web console and create client credentials there
- relay TCP ports you hand out via `-remoteTCPPort` must also be published (`-p 6022:6022`, or a range)

Prefer a plain binary? A minimal systemd unit:

```ini
# /etc/systemd/system/gt-server.service
[Unit]
Description=G-Tunnel relay
After=network-online.target

[Service]
ExecStart=/usr/local/bin/gt server -c /etc/gt/server.yaml
Restart=always

[Install]
WantedBy=multi-user.target
```

A matching `server.yaml` with users and quotas is shown in [multi-user relay](#multi-user-relay-with-quotas); the full option list is in [Configuration](config.md#server-options).

## Multi-user relay with quotas

Register users in the server config, each with its own credentials and limits:

```yaml
# server.yaml
addr: "7000"
webAddr: "127.0.0.1:8000"

users:
  alice:
    secret: alices-secret
    tcp:
      - range: 30000-39999   # pool she may draw -remoteTCPPort/-remoteTCPRandom from
    tcpNumber: 4             # max ports she can open
    speed: 10485760          # bytes/sec
    connections: 10          # max tunnel connections
  bob:
    secret: bobs-secret
    host:
      number: 2              # max host-prefix services
      withID: true           # his prefixes are namespaced as bob-<prefix>
```

```bash
gt server -c ./server.yaml
```

For a quick setup without a file, repeat `-id`/`-secret` pairs on the command line:

```bash
gt server -addr 7000 -id alice -secret alices-secret -id bob -secret bobs-secret
```

!> `allowAnyClient: true` turns the relay into an open proxy for anyone who knows the address — never enable it on a public server.

Changed the users file? Apply it without dropping existing connections:

```bash
gt -s reload
```

Reload re-reads the config and keeps forwarding alive — safe to run while traffic flows.
