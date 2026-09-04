# Configuration

`gt` reads configuration from two places, later winning:

1. a YAML config file — `-c ./config.yaml`, or the default path that the web console writes to
2. command-line flags

Point `-c` at a **directory** to batch-start several roles at once — each file must declare its role with a top-level `type: client`, `type: server` (or contain a `services:` key, which implies client):

```bash
gt -c ./conf.d
```

Apply changes of a running instance without dropping connections:

```bash
gt -s reload    # also: restart, stop
```

Reload re-reads the config; work processes keep forwarding during the switch. The same mechanism lets you upgrade the binary under load with zero downtime.

## Client options

```yaml
# client.yaml
id: myapp
secret: s3cret
remote: tcp://relay.example.com:7000

services:
  - hostPrefix: web
    local: http://127.0.0.1:8080
```

| Option (yaml / flag) | Default | Description |
|---|---|---|
| `id` / `-id` | — | Client id; also the default host prefix. 1–200 chars |
| `secret` / `-secret` | — | Secret paired with the id. 1–200 chars |
| `remote` / `-remote` | `tcp://` | Relay address; repeatable. Schemes: `tcp://`, `tls://`, `quic://`. The client probes the network and picks the working scheme automatically when several are given |
| `reconnectDelay` | `5s` | Delay before reconnect attempts |
| `remoteConnections` | `3` (1–10) | Max connections kept in the relay pool |
| `remoteIdleConnections` | `1` | Idle connections kept warm |
| `remoteTimeout` | `45s` | Relay connection timeout |
| `remoteSTUN` / `-remoteSTUN` | — | STUN server for WebRTC P2P, e.g. `stun:1.2.3.4:3478` |
| `remoteCert` / `-remoteCert` | — | Path to the relay's certificate (pinning) |
| `remoteCertInsecure` | `false` | Accept self-signed relay certificates |
| `webAddr` / `-webAddr` | off | Web console address, e.g. `127.0.0.1:7000` |
| `logLevel` / `-logLevel` | `info` | `trace`…`panic`, or `disable` |
| `logFile` + `logFileMaxSize` (512MB) + `logFileMaxCount` (7) | off | Rotated file logging |
| `bbr` | `false` | BBR congestion control for QUIC (both ends must enable) |

Per-service entries under `services:` (positional flags `-hostPrefix`, `-local`, `-remoteTCPPort`, `-remoteTCPRandom`, `-useLocalAsHTTPHost`, `-localTimeout` mirror them):

| Field | Description |
|---|---|
| `hostPrefix` | Hostname prefix this service answers to |
| `local` | Local service URL: `http://`, `https://` or `tcp://host:port` |
| `remoteTCPPort` | Fixed TCP port the relay opens for this service |
| `remoteTCPRandom` | Let the relay assign a port from its configured range |
| `localTimeout` | Timeout for local connections |
| `useLocalAsHTTPHost` | Pass the local address as HTTP `Host` to the backend |

WebRTC / P2P extras: `webrtcConnections` (max P2P connections, default 30), `webrtcConnectionIdleTimeout` (5m), `webrtcMinPort`/`webrtcMaxPort` (UDP port range), `webrtcThreadMode`. TCP-forward (gateway) extras: `tcpForwardAddr`, `tcpForwardHostPrefix`, `tcpForwardConnections`.

## Server options

```yaml
# server.yaml
addr: "7000"
webAddr: "127.0.0.1:8000"
```

| Option (yaml) | Default | Description |
|---|---|---|
| `addr` / `-addr` | — | Relay listener, e.g. `80`, `:80`, `0.0.0.0:80` |
| `tlsAddr` + `certFile` + `keyFile` | off | TLS listener for `tls://` clients (`tlsVersion` min is `tls1.2`) |
| `quicAddr` + `certFile` + `keyFile` | off | QUIC listener for `quic://` clients |
| `stunAddr` / `-stunAddr` | off | STUN listener (UDP 3478) for WebRTC P2P |
| `sniAddr` | off | Raw TLS forwarder routing by SNI — end-to-end encrypted passthrough |
| `apiAddr` | off | Internal API listener |
| `webAddr` / `-webAddr` | off | Web console, e.g. `0.0.0.0:8000` (add `admin`/`password`/`signingKey` for auth) |
| `users:` or repeated `-id`/`-secret` | — | Client credentials; see [multi-user example](examples.md#multi-user-relay-with-quotas) |
| `users[].tcp[].range` | — | Port pool a user may open, e.g. `30000-39999` |
| `users[].tcpNumber` | — | Max ports per user |
| `users[].speed` | unlimited | Per-user rate limit, bytes/sec |
| `users[].connections` | `10` | Max tunnel connections per user |
| `host.number` | `0` | Max host-prefix services per user |
| `host.regex` | — | Allowed prefix patterns |
| `host.withID` | `false` | Namespace prefixes as `id-host` |
| `timeout` | `90s` | Idle connection timeout (`timeoutOnUnidirectionalTraffic` tightens it) |
| `httpMUXHeader` | `Host` | Header used for host-prefix routing |
| `allowAnyClient` | `false` | Accept any id/secret — **never on public servers** |
| `bbr` | `false` | BBR congestion control for QUIC |
| `logLevel`, `logFile`, `logFileMaxSize`, `logFileMaxCount` | `info` | Logging, same as client |

`authAPI` / `usageAPI` let an external service validate credentials and collect per-user traffic stats — that is how the hosted console integrates.
